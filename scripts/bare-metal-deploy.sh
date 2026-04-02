#!/bin/bash
#==============================================================================
# VIMI Bare Metal Deployment Script
# Deploys VIMI mission processing system on Ubuntu 22.04+
#
# Usage:
#   sudo ./bare-metal-deploy.sh deploy     # Full deployment
#   sudo ./bare-metal-deploy.sh infra       # Infrastructure only
#   sudo ./bare-metal-deploy.sh vimi       # VIMI services only
#   sudo ./bare-metal-deploy.sh verify      # Verify deployment
#   sudo ./bare-metal-deploy.sh start       # Start all services
#   sudo ./bare-metal-deploy.sh stop        # Stop all services
#   sudo ./bare-metal-deploy.sh restart      # Restart all services
#   sudo ./bare-metal-deploy.sh clean       # Remove all services
#
# Requirements:
#   - Ubuntu 22.04 LTS or later
#   - Root/sudo access
#   - Internet access (for downloads)
#==============================================================================

set -euo pipefail

# Configuration
readonly VIMI_USER="vimi"
readonly VIMI_HOME="/opt/vimi"
readonly VIMI_DATA="/var/data/vimi"
readonly VIMI_LOG="/var/log/vimi"
readonly KAFKA_VERSION="3.7.0"
readonly KAFKA_DIR="/opt/kafka_${KAFKA_VERSION}"
readonly ETCD_VERSION="v3.5.20"
readonly GO_VERSION="1.22.12"
readonly KAFKA_PORT=9092
readonly ETCD_PORT=2379
readonly REDIS_PORT=6379
readonly POSTGRES_PORT=5432

# VIMI service ports
readonly VIMI_PORTS=(
    "opir-ingest:8080"
    "missile-warning-engine:8080"
    "sensor-fusion:8082"
    "lvc-coordinator:8083"
    "alert-dissemination:8084"
    "env-monitor:8085"
    "replay-engine:8086"
    "data-catalog:8087"
    "dis-hla-gateway:8090"
    "vimi-plugin:8091"
)

# Kafka topics
readonly KAFKA_TOPICS=(
    "vimi.opir.sensor-data:3:1"
    "vimi.tracks:3:1"
    "vimi.fusion.tracks:3:1"
    "vimi.alerts:3:1"
    "vimi.c2.alerts:3:1"
    "vimi.alert-log:3:1"
    "vimi.dis.entity-state:3:1"
    "vimi.dis.entity-state-out:3:1"
    "vimi.hla.object-update:3:1"
    "vimi.hla.interaction:3:1"
    "vimi.env.events:3:1"
    "vimi.replay.events:3:1"
)

# Colors
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

#==============================================================================
# Logging functions
#==============================================================================
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step() { echo -e "\n${BLUE}==>${NC} ${GREEN}$*${NC}"; }

#==============================================================================
# Check functions
#==============================================================================
is_root() { [[ ${EUID:-$(id -e)} -eq 0 ]]; }
is_ubuntu() { grep -q 'Ubuntu' /etc/os-release 2>/dev/null; }

#==============================================================================
# User management
#==============================================================================
create_vimi_user() {
    log_step "Creating VIMI user"
    if id "$VIMI_USER" &>/dev/null; then
        log_ok "User $VIMI_USER already exists"
    else
        useradd -r -s /bin/bash -m -d "$VIMI_HOME" "$VIMI_USER"
        log_ok "Created user: $VIMI_USER"
    fi
}

create_directories() {
    log_step "Creating directories"
    mkdir -p "$VIMI_HOME"/{bin,config,data,logs,services}
    mkdir -p "$VIMI_DATA"/{kafka,etcd,redis,postgres}
    mkdir -p "$VIMI_LOG"
    chown -R "$VIMI_USER:$VIMI_USER" "$VIMI_HOME" "$VIMI_DATA" "$VIMI_LOG"
    log_ok "Directories created"
}

#==============================================================================
# Go installation
#==============================================================================
install_go() {
    log_step "Installing Go ${GO_VERSION}"
    
    if /usr/local/go/bin/go version &>/dev/null 2>&1; then
        local current_ver=$(/usr/local/go/bin/go version | grep -oP 'go\K[0-9]+\.[0-9]+')
        if [[ "$current_ver" == "1.22" ]]; then
            log_ok "Go already installed"
            return 0
        fi
    fi
    
    local arch=$(uname -m)
    case $arch in
        x86_64) local arch_name="amd64" ;;
        aarch64) local arch_name="arm64" ;;
        *) log_error "Unsupported architecture: $arch"; return 1 ;;
    esac
    
    cd /tmp
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${arch_name}.tar.gz" -O go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go.tar.gz
    rm go.tar.gz
    
    cat > /etc/profile.d/go.sh << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
EOF
    source /etc/profile.d/go.sh
    
    log_ok "Go installed: $(go version)"
}

#==============================================================================
# Kafka installation (KRaft mode, no Zookeeper)
#==============================================================================
install_kafka() {
    log_step "Installing Kafka ${KAFKA_VERSION}"
    
    if [[ -d "$KAFKA_DIR" ]]; then
        log_ok "Kafka already installed"
        return 0
    fi
    
    cd /tmp
    wget -q "https://downloads.apache.org/kafka/${KAFKA_VERSION}/kafka_2.13-${KAFKA_VERSION}.tgz" -O kafka.tgz
    tar -xzf kafka.tgz -C /opt
    rm kafka.tgz
    ln -s "$KAFKA_DIR" /opt/kafka
    
    # Configure Kafka for KRaft mode
    cat > /opt/kafka/config/kraft/server.properties << 'EOF'
process.roles=controller,broker
node.id=1
controller.quorum.voters=1@localhost:9093
inter.broker.listener.name=PLAINTEXT
listeners=PLAINTEXT://:9092,CONTROLLER://:9093
advertised.listeners=PLAINTEXT://localhost:9092
listener.security.protocol.map=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
controller.listener.names=CONTROLLER
log.dirs=/var/data/vimi/kafka
num.partitions=3
default.replication.factor=1
min.insync.replicas=1
auto.create.topics.enable=true
compression.type=producer
EOF
    
    log_ok "Kafka installed"
}

setup_kafka_service() {
    log_step "Setting up Kafka systemd service"
    
    cat > /etc/systemd/system/kafka.service << 'EOF'
[Unit]
Description=Apache Kafka (KRaft)
Documentation=https://kafka.apache.org/
After=network.target

[Service]
Type=simple
User=vimi
ExecStart=/opt/kafka/bin/kafka-server-start.sh /opt/kafka/config/kraft/server.properties
ExecStop=/opt/kafka/bin/kafka-server-stop.sh
Restart=on-failure
RestartSec=10
LimitNOFILE=65536
TimeoutStartSec=300
TimeoutStopSec=300

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-realmodeload
    systemctl daemon-reload
    log_ok "Kafka service created"
}

#==============================================================================
# etcd installation
#==============================================================================
install_etcd() {
    log_step "Installing etcd ${ETCD_VERSION}"
    
    local etcd_bin="/opt/etcd/bin/etcd"
    if [[ -x "$etcd_bin" ]]; then
        log_ok "etcd already installed"
        return 0
    fi
    
    cd /tmp
    wget -q "https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz" -O etcd.tar.gz
    mkdir -p /opt/etcd/bin
    tar -xzf etcd.tar.gz -C /opt/etcd --strip-components=1
    rm etcd.tar.gz
    
    cat > /etc/systemd/system/etcd.service << 'EOF'
[Unit]
Description=etcd distributed key-value store
Documentation=https://etcd.io/
After=network.target

[Service]
Type=simple
User=vimi
ExecStart=/opt/etcd/bin/etcd \
    --data-dir=/var/data/vimi/etcd \
    --listen-client-urls=http://0.0.0.0:2379 \
    --advertise-client-urls=http://localhost:2379 \
    --listen-peer-urls=http://0.0.0.0:2380 \
    --initial-advertise-peer-urls=http://localhost:2380 \
    --initial-cluster=default=http://localhost:2380 \
    --name=default
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    log_ok "etcd installed"
}

#==============================================================================
# Redis installation
#==============================================================================
install_redis() {
    log_step "Installing Redis"
    
    apt-get install -y redis-server
    
    # Configure Redis for VIMI
    cat > /etc/redis/redis.conf << 'EOF'
bind 0.0.0.0
port 6379
protected-mode no
maxmemory 512mb
maxmemory-policy allkeys-lru
save ""
appendonly no
timeout 300
tcp-keepalive 60
EOF
    
    systemctl restart redis
    systemctl enable redis
    log_ok "Redis installed"
}

#==============================================================================
# PostgreSQL installation
#==============================================================================
install_postgres() {
    log_step "Installing PostgreSQL"
    
    apt-get install -y postgresql postgresql-contrib
    
    # Create VIMI database and user
    sudo -u postgres psql << 'EOF'
CREATE USER vimi WITH PASSWORD 'vimi_secure_password';
CREATE DATABASE vimi OWNER vimi;
GRANT ALL PRIVILEGES ON DATABASE vimi TO vimi;
\connect vimi;
GRANT ALL ON SCHEMA public TO vimi;
EOF
    
    # Allow password auth locally
    echo "host all all 127.0.0.1/32 md5" >> /etc/postgresql/*/main/pg_hba.conf
    echo "host all all ::1/128 md5" >> /etc/postgresql/*/main/pg_hba.conf
    
    systemctl restart postgresql
    systemctl enable postgresql
    log_ok "PostgreSQL installed"
}

#==============================================================================
# Infrastructure deployment
#==============================================================================
deploy_infrastructure() {
    log_step "Deploying infrastructure"
    
    create_vimi_user
    create_directories
    install_go
    install_kafka
    setup_kafka_service
    install_etcd
    install_redis
    install_postgres
    
    # Format Kafka storage (one-time)
    if [[ ! -f /var/data/vimi/kafka/meta.properties ]]; then
        log_info "Formatting Kafka storage..."
        local cluster_id=$(/opt/kafka/bin/kafka-storage.sh random-uuid)
        /opt/kafka/bin/kafka-storage.sh format \
            -t "$cluster_id" \
            -c /opt/kafka/config/kraft/server.properties \
            --ignore-formatted
        chown -R vimi:vimi /var/data/vimi/kafka
        log_ok "Kafka storage formatted"
    fi
    
    # Start infrastructure services
    systemctl enable etcd redis postgresql kafka
    systemctl start etcd redis postgresql
    
    # Wait for Kafka to be ready
    log_info "Waiting for Kafka to be ready..."
    sleep 15
    
    # Start Kafka
    if systemctl is-active kafka &>/dev/null; then
        log_ok "Kafka is running"
    else
        log_warn "Kafka may not be fully ready yet"
        systemctl start kafka
        sleep 10
    fi
    
    log_ok "Infrastructure deployed"
}

#==============================================================================
# Build VIMI services
#==============================================================================
build_vimi_services() {
    log_step "Building VIMI services"
    
    if [[ ! -d /opt/vimi ]]; then
        log_error "VIMI source not found at /opt/vimi"
        log_info "Please clone the repository first:"
        log_info "  git clone https://github.com/vimic/trooper-vimi.git /opt/vimi"
        return 1
    fi
    
    cd /opt/vimi
    
    # Build each service
    for entry in "${VIMI_PORTS[@]}"; do
        local svc="${entry%%:*}"
        local port="${entry##*:}"
        
        log_info "Building $svc..."
        
        case $svc in
            dis-hla-gateway)
                if [[ -f hla-bridge/main.go ]]; then
                    (cd hla-bridge && /usr/local/go/bin/go build -o ../bin/$svc .) || true
                fi
                ;;
            vimi-plugin)
                if [[ -f vimi-plugin/main.go ]]; then
                    (cd vimi-plugin && /usr/local/go/bin/go build -o ../bin/$svc .) || true
                fi
                ;;
            *)
                if [[ -f apps/$svc/main.go ]]; then
                    (cd apps/$svc && /usr/local/go/bin/go build -o ../../bin/$svc .) || true
                fi
                ;;
        esac
        
        if [[ -f /opt/vimi/bin/$svc ]]; then
            log_ok "  $svc built"
        else
            log_warn "  $svc build may have failed"
        fi
    done
    
    chown -R vimi:vimi /opt/vimi/bin
    log_ok "VIMI services built"
}

#==============================================================================
# Create VIMI systemd services
#==============================================================================
create_vimi_services() {
    log_step "Creating VIMI systemd services"
    
    for entry in "${VIMI_PORTS[@]}"; do
        local svc="${entry%%:*}"
        local port="${entry##*:}"
        
        cat > /etc/systemd/system/vimi-${svc}.service << EOF
[Unit]
Description=VIMI ${svc}
After=network.target kafka.service etcd.service redis.service postgresql.service
Wants=kafka.service etcd.service redis.service postgresql.service

[Service]
Type=simple
User=$VIMI_USER
WorkingDirectory=$VIMI_HOME/services/$svc
Environment="PORT=$port"
Environment="KAFKA_BROKERS=localhost:${KAFKA_PORT}"
Environment="ETCD_ENDPOINTS=localhost:${ETCD_PORT}"
Environment="REDIS_ADDR=localhost:${REDIS_PORT}"
Environment="POSTGRES_URL=postgres://vimi:vimi_secure_password@localhost:${POSTGRES_PORT}/vimi"
Environment="LOG_LEVEL=info"
ExecStart=$VIMI_HOME/bin/$svc
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vimi-$svc

[Install]
WantedBy=multi-user.target
EOF
    done
    
    # Create convenience target for all services
    cat > /etc/systemd/system/vimi.target << 'EOF'
[Unit]
Description=VIMI Mission Processing System
After=network.target kafka.service etcd.service redis.service postgresql.service
Wants=kafka.service etcd.service redis.service postgresql.service
Before=vimi-opir-ingest.service vimi-missile-warning-engine.service

[Install]
Also=vimi-opir-ingest.service
Also=vimi-missile-warning-engine.service
Also=vimi-sensor-fusion.service
Also=vimi-lvc-coordinator.service
Also=vimi-alert-dissemination.service
Also=vimi-env-monitor.service
Also=vimi-replay-engine.service
Also=vimi-data-catalog.service
Also=vimi-dis-hla-gateway.service
Also=vimi-vimi-plugin.service
EOF
    
    systemctl daemon-reload
    log_ok "VIMI services created"
}

#==============================================================================
# Deploy VIMI
#==============================================================================
deploy_vimi() {
    log_step "Deploying VIMI services"
    
    # Reload daemon since we may have created new units
    systemctl daemon-reload
    
    # Enable all services
    for entry in "${VIMI_PORTS[@]}"; do
        local svc="${entry%%:*}"
        systemctl enable "vimi-${svc}.service"
    done
    
    # Create Kafka topics
    create_kafka_topics
    
    # Start services in dependency order
    systemctl start vimi-opir-ingest
    sleep 2
    systemctl start vimi-missile-warning-engine
    sleep 2
    systemctl start vimi-sensor-fusion
    sleep 2
    systemctl start vimi-lvc-coordinator
    sleep 1
    systemctl start vimi-alert-dissemination
    sleep 1
    systemctl start vimi-env-monitor
    sleep 1
    systemctl start vimi-replay-engine
    sleep 1
    systemctl start vimi-data-catalog
    sleep 1
    systemctl start vimi-dis-hla-gateway
    sleep 1
    systemctl start vimi-vimi-plugin
    
    log_ok "VIMI services deployed"
}

#==============================================================================
# Create Kafka topics
#==============================================================================
create_kafka_topics() {
    log_step "Creating Kafka topics"
    
    if ! systemctl is-active kafka &>/dev/null; then
        log_warn "Kafka not running, skipping topic creation"
        return 1
    fi
    
    local kafka_bin="/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:${KAFKA_PORT}"
    
    for spec in "${KAFKA_TOPICS[@]}"; do
        local topic="${spec%%:*}"
        local parts="${spec%%:*}"
        local replicas="${spec##*:}"
        
        $kafka_bin --create --topic "$topic" \
            --partitions "$parts" \
            --replication-factor "$replicas" \
            --if-not-exists &>/dev/null
        echo -n "."
    done
    echo
    
    log_ok "Kafka topics created"
}

#==============================================================================
# Deploy everything
#==============================================================================
full_deploy() {
    log_step "Starting full VIMI deployment"
    
    if ! is_root; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
    
    if ! is_ubuntu; then
        log_warn "This script is designed for Ubuntu. Other distros may work but are untested."
    fi
    
    deploy_infrastructure
    build_vimi_services
    create_vimi_services
    deploy_vimi
    verify_deployment
    
    log_step "Deployment complete!"
    echo ""
    echo "VIMI is now deployed and running."
    echo "Access the services at:"
    echo "  - opir-ingest:        http://localhost:8080"
    echo "  - missile-warning:    http://localhost:8080"
    echo "  - sensor-fusion:     http://localhost:8082"
    echo "  - lvc-coordinator:    http://localhost:8083"
    echo "  - alert-dissemination: http://localhost:8084"
    echo "  - env-monitor:        http://localhost:8085"
    echo "  - replay-engine:      http://localhost:8086"
    echo "  - data-catalog:       http://localhost:8087"
    echo "  - dis-hla-gateway:    http://localhost:8090"
    echo "  - vimi-plugin:        http://localhost:8091"
    echo ""
    echo "Manage services with:"
    echo "  systemctl status 'vimi-*'"
    echo "  journalctl -u vimi-opir-ingest -f"
}

#==============================================================================
# Verify deployment
#==============================================================================
verify_deployment() {
    log_step "Verifying deployment"
    
    local failures=0
    
    echo ""
    echo "Checking VIMI services..."
    for entry in "${VIMI_PORTS[@]}"; do
        local svc="${entry%%:*}"
        local port="${entry##*:}"
        
        if systemctl is-active "vimi-${svc}" &>/dev/null; then
            if curl -sf "http://localhost:${port}/health" &>/dev/null; then
                log_ok "$svc (:$port)"
            else
                log_warn "$svc (:$port) - service running but health check failed"
            fi
        else
            log_error "$svc (:$port) - NOT running"
            ((failures++))
        fi
    done
    
    echo ""
    echo "Checking infrastructure..."
    
    if systemctl is-active kafka &>/dev/null; then
        log_ok "Kafka"
    else
        log_error "Kafka - NOT running"
        ((failures++))
    fi
    
    if systemctl is-active etcd &>/dev/null; then
        log_ok "etcd"
    else
        log_error "etcd - NOT running"
        ((failures++))
    fi
    
    if systemctl is-active redis &>/dev/null; then
        log_ok "Redis"
    else
        log_error "Redis - NOT running"
        ((failures++))
    fi
    
    if systemctl is-active postgresql &>/dev/null; then
        log_ok "PostgreSQL"
    else
        log_error "PostgreSQL - NOT running"
        ((failures++))
    fi
    
    echo ""
    if [[ $failures -eq 0 ]]; then
        log_ok "All systems operational"
    else
        log_error "$failures system(s) failed"
    fi
    
    return $failures
}

#==============================================================================
# Service management
#==============================================================================
start_services() {
    log_step "Starting VIMI services"
    systemctl start kafka etcd redis postgresql
    sleep 5
    systemctl start vimi.target
    systemctl start 'vimi-@all' 2>/dev/null || \
        systemctl start vimi-{opir-ingest,missile-warning-engine,sensor-fusion,lvc-coordinator,alert-dissemination,env-monitor,replay-engine,data-catalog,dis-hla-gateway,vimi-plugin}
    log_ok "Services started"
}

stop_services() {
    log_step "Stopping VIMI services"
    systemctl stop vimi.target 2>/dev/null || true
    systemctl stop 'vimi-*' 2>/dev/null || true
    log_ok "Services stopped"
}

restart_services() {
    stop_services
    sleep 2
    start_services
}

#==============================================================================
# Clean deployment
#==============================================================================
clean_deployment() {
    log_warn "This will remove ALL VIMI services and data!"
    read -p "Are you sure? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Aborted"
        return
    fi
    
    stop_services
    
    rm -f /etc/systemd/system/vimi-*.service
    rm -f /etc/systemd/system/vimi.target
    systemctl daemon-reload
    
    rm -rf "$VIMI_HOME" "$VIMI_DATA" "$VIMI_LOG"
    rm -rf /opt/kafka /opt/etcd
    
    log_ok "Cleaned"
}

#==============================================================================
# Main
#==============================================================================
main() {
    local command="${1:-deploy}"
    
    case $command in
        deploy|full)
            full_deploy
            ;;
        infra|infrastructure)
            deploy_infrastructure
            ;;
        vimi)
            build_vimi_services
            create_vimi_services
            deploy_vimi
            ;;
        verify|check|status)
            verify_deployment
            ;;
        start)
            if ! is_root; then log_error "Need root"; exit 1; fi
            start_services
            ;;
        stop)
            if ! is_root; then log_error "Need root"; exit 1; fi
            stop_services
            ;;
        restart)
            if ! is_root; then log_error "Need root"; exit 1; fi
            restart_services
            ;;
        clean|purge)
            if ! is_root; then log_error "Need root"; exit 1; fi
            clean_deployment
            ;;
        topics)
            create_kafka_topics
            ;;
        *)
            echo "Usage: sudo $0 {deploy|infra|vimi|verify|start|stop|restart|clean|topics}"
            echo ""
            echo "Commands:"
            echo "  deploy   - Full deployment (default)"
            echo "  infra    - Infrastructure only (Kafka, etcd, Redis, PostgreSQL)"
            echo "  vimi     - VIMI services only"
            echo "  verify   - Verify deployment status"
            echo "  start    - Start all services"
            echo "  stop     - Stop all services"
            echo "  restart  - Restart all services"
            echo "  clean    - Remove all VIMI services and data"
            echo "  topics   - Create Kafka topics"
            exit 1
            ;;
    esac
}

main "$@"
