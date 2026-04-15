// Package ha provides leader election support
package ha

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ElectionType defines election backend type
type ElectionType string

const (
	ElectionEtcd  ElectionType = "etcd"
	ElectionRedis ElectionType = "redis"
)

// LeaderElection provides leader election
type LeaderElection struct {
	mu            sync.RWMutex
	leader        string
	term          uint64
	electionType  ElectionType
	namespace     string
	instanceID    string
	leaseDuration time.Duration
	renewInterval time.Duration
	isLeader      bool
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// Config configures leader election
type Config struct {
	ElectionType  ElectionType
	Namespace     string
	InstanceID    string
	LeaseDuration time.Duration
	RenewInterval time.Duration
}

// NewLeaderElection creates a new leader election
func NewLeaderElection(cfg Config) *LeaderElection {
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = 15 * time.Second
	}
	if cfg.RenewInterval == 0 {
		cfg.RenewInterval = 5 * time.Second
	}

	return &LeaderElection{
		electionType:  cfg.ElectionType,
		namespace:     cfg.Namespace,
		instanceID:    cfg.InstanceID,
		leaseDuration: cfg.LeaseDuration,
		renewInterval: cfg.RenewInterval,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Campaign starts election campaign
func (le *LeaderElection) Campaign(ctx context.Context) error {
	switch le.electionType {
	case ElectionEtcd:
		return le.campaignEtcd(ctx)
	case ElectionRedis:
		return le.campaignRedis(ctx)
	default:
		return fmt.Errorf("unknown election type: %s", le.electionType)
	}
}

// campaignEtcd campaigns using etcd
func (le *LeaderElection) campaignEtcd(ctx context.Context) error {
	// In production, use etcd client for leader election
	// etcd concurrency package provides session and election support

	le.mu.Lock()
	le.isLeader = true
	le.leader = le.instanceID
	le.term++
	le.mu.Unlock()

	// Start renewal goroutine
	go le.renewEtcdLease()

	return nil
}

// campaignRedis campaigns using Redis
func (le *LeaderElection) campaignRedis(ctx context.Context) error {
	// In production, use Redis SETNX with expiration for leader election

	le.mu.Lock()
	le.isLeader = true
	le.leader = le.instanceID
	le.term++
	le.mu.Unlock()

	// Start renewal goroutine
	go le.renewRedisLease()

	return nil
}

// renewEtcdLease renews etcd lease
func (le *LeaderElection) renewEtcdLease() {
	ticker := time.NewTicker(le.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// In production, renew etcd session
		case <-le.stopCh:
			return
		}
	}
}

// renewRedisLease renews Redis lease
func (le *LeaderElection) renewRedisLease() {
	ticker := time.NewTicker(le.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// In production, extend Redis key TTL
		case <-le.stopCh:
			return
		}
	}
}

// Resign resigns from leadership
func (le *LeaderElection) Resign(ctx context.Context) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	if !le.isLeader {
		return nil
	}

	le.isLeader = false
	le.leader = ""
	close(le.stopCh)

	return nil
}

// IsLeader returns whether this instance is leader
func (le *LeaderElection) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.isLeader
}

// GetLeader returns current leader
func (le *LeaderElection) GetLeader() string {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.leader
}

// GetTerm returns current term
func (le *LeaderElection) GetTerm() uint64 {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.term
}

// OnLeadershipAcquired registers callback for leadership acquisition
func (le *LeaderElection) OnLeadershipAcquired(callback func()) {
	// In production, call when becoming leader
}

// OnLeadershipLost registers callback for leadership loss
func (le *LeaderElection) OnLeadershipLost(callback func()) {
	// In production, call when losing leadership
}

// EtcdElection implements etcd-based election
type EtcdElection struct {
	endpoints  []string
	namespace  string
	instanceID string
	ttl        int
}

// NewEtcdElection creates etcd election
func NewEtcdElection(endpoints []string, namespace, instanceID string, ttl int) *EtcdElection {
	return &EtcdElection{
		endpoints:  endpoints,
		namespace:  namespace,
		instanceID: instanceID,
		ttl:        ttl,
	}
}

// Campaign campaigns for leadership
func (e *EtcdElection) Campaign(ctx context.Context) error {
	// In production: use clientv3/concurrency package
	// session, err := concurrency.NewSession(client, concurrency.WithTTL(e.ttl))
	// election := concurrency.NewElection(session, e.namespace)
	// err := election.Campaign(ctx, e.instanceID)
	return nil
}

// Resign resigns leadership
func (e *EtcdElection) Resign(ctx context.Context) error {
	// In production: election.Resign(ctx)
	return nil
}

// Observe observes leadership changes
func (e *EtcdElection) Observe(ctx context.Context) <-chan string {
	ch := make(chan string)
	// In production: election.Observe(ctx)
	return ch
}

// RedisElection implements Redis-based election
type RedisElection struct {
	addr       string
	namespace  string
	instanceID string
	ttl        time.Duration
}

// NewRedisElection creates Redis election
func NewRedisElection(addr, namespace, instanceID string, ttl time.Duration) *RedisElection {
	return &RedisElection{
		addr:       addr,
		namespace:  namespace,
		instanceID: instanceID,
		ttl:        ttl,
	}
}

// Campaign campaigns for leadership
func (r *RedisElection) Campaign(ctx context.Context) error {
	// In production: use SETNX with NX and EX options
	// result, err := redis.SetNX(ctx, key, value, ttl)
	return nil
}

// Resign resigns leadership
func (r *RedisElection) Resign(ctx context.Context) error {
	// In production: DEL key
	return nil
}

// Extend extends the lease
func (r *RedisElection) Extend(ctx context.Context) error {
	// In production: EXPIRE key ttl
	return nil
}
