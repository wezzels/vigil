// vimic-plugin — TROOPER-VIMI VIMIC Integration Module
// Extends VIMIC with VIMI mission processing capabilities
//
// This plugin adds VIMI-specific functionality to VIMIC:
// - VIMI VM template management (SBIRS, C2, sensor nodes)
// - DIS protocol entity management via VIMIC UI
// - Federation discovery and connection
// - Alert dashboard integration
//
// Build: go build -o vimic-plugin.so -buildmode=plugin .
package vimicplugin

import (
    "encoding/xml"
    "fmt"
    "net/url"
    "time"
)

// VIMIComponent represents a VIMI mission app deployed on a VM
type VIMIComponent struct {
    Name           string    // e.g. "opir-ingest-01"
    Type           string    // e.g. "opir-ingest", "missile-warning"
    VMRef          string    // VIMIC VM name
    Site           uint16    // DIS Site ID
    Application    uint16    // DIS Application ID
    KafkaTopic     string    // Kafka topic for mission events
    Status         string    // "running", "stopped", "degraded"
    LastHeartbeat  time.Time
}

// VIMIFederation represents a DIS/HLA federation
type VIMIFederation struct {
    Name           string    // e.g. "TROOPER-VIMI-RTO-01"
    Type           string    // "DIS", "HLA", "TENA", "NETN"
    MulticastGroup string    // e.g. "239.255.0.1"
    Port           uint16    // DIS port (default 3000)
    ExerciseID     uint8     // DIS exercise ID
    Participants   []string  // Site IDs of participants
}

// AlertLevel from VIMI alert system
type AlertLevel int

const (
    ALERT_UNKNOWN   AlertLevel = 0
    ALERT_CONOPREP  AlertLevel = 1  // CONOPREP: preparations detected
    ALERT_IMMINENT  AlertLevel = 2  // IMMINENT: launch detected
    ALERT_INCOMING  AlertLevel = 3  // INCOMING: missile in-flight
    ALERT_HOSTILE   AlertLevel = 4  // HOSTILE: impact imminent
)

// MissileAlert represents a VIMI missile warning alert
type MissileAlert struct {
    AlertID         uint32
    Level           AlertLevel
    ThreatType      string  // "SRBM", "IRBM", "ICBM", etc.
    LaunchLocation  [3]float64  // lat, lon, alt (deg, deg, m)
    ImpactLocation  [3]float64
    TimeToImpact   float32  // seconds
    TrackNumber     uint32
    SensorSource    string  // "SBIRS_HIGH", "NG_OPIR", etc.
    NCARequired     bool
    IssuedAt       time.Time
}

// VIMICPlugin extends VIMIC with VIMI capabilities
type VIMICPlugin struct {
    Components map[string]*VIMIComponent
    Federations map[string]*VIMIFederation
    Alerts     []*MissileAlert
}

// New creates a new VIMI VIMIC plugin
func New() *VIMICPlugin {
    return &VIMICPlugin{
        Components: make(map[string]*VIMIComponent),
        Federations: make(map[string]*VIMIFederation),
        Alerts: []*MissileAlert{},
    }
}

// RegisterComponent adds a VIMI component to the plugin
func (p *VIMICPlugin) RegisterComponent(c *VIMIComponent) error {
    if c.Name == "" || c.Type == "" {
        return fmt.Errorf("component name and type required")
    }
    p.Components[c.Name] = c
    return nil
}

// GetComponent returns a VIMI component by name
func (p *VIMICPlugin) GetComponent(name string) *VIMIComponent {
    return p.Components[name]
}

// ListComponents returns all registered components
func (p *VIMICPlugin) ListComponents() []*VIMIComponent {
    result := make([]*VIMIComponent, 0, len(p.Components))
    for _, c := range p.Components {
        result = append(result, c)
    }
    return result
}

// CreateVIMIVM creates a VIMI VM template XML for VIMIC
func (p *VIMICPlugin) CreateVIMIVM(template string) (string, error) {
    templates := map[string]string{
        "sbirs-sensor":    VIMISBIRSVMXML,
        "c2-node":         VIMIC2VMXML,
        "alert-processor": VIMIAlertVMXML,
        "replay-server":   VIMIReplayVMXML,
    }
    xml, ok := templates[template]
    if !ok {
        return "", fmt.Errorf("unknown template: %s", template)
    }
    return xml, nil
}

// StartFederation initiates a DIS federation
func (p *VIMICPlugin) StartFederation(f *VIMIFederation) error {
    if f.MulticastGroup == "" {
        f.MulticastGroup = "239.255.0.1"
    }
    if f.Port == 0 {
        f.Port = 3000
    }
    p.Federations[f.Name] = f
    return nil
}

// GetAlerts returns current missile alerts
func (p *VIMICPlugin) GetAlerts() []*MissileAlert {
    return p.Alerts
}

// RecordAlert adds a new missile alert
func (p *VIMICPlugin) RecordAlert(a *MissileAlert) {
    a.IssuedAt = time.Now()
    p.Alerts = append(p.Alerts, a)
}

// VIMI VM Templates (minimal cloud-init enabled)
const VIMISBIRSVMXML = `<domain type='kvm'>
  <name>vimi-sbirs-sensor</name>
  <memory unit='MiB'>8192</memory>
  <vcpu>4</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <q35/>
  <cpu mode='host-passthrough'/>
  <interface type='network'>
    <source network='default'/>
    <model type='virtio'/>
  </interface>
  <disk type='file' device='disk'>
    <driver name='qemu' type='qcow2'/>
    <source file='/var/lib/libvirt/images/vimi-sbirs.qcow2'/>
    <target dev='vda' bus='virtio'/>
  </disk>
</domain>`

const VIMIC2VMXML = `<domain type='kvm'>
  <name>vimi-c2-node</name>
  <memory unit='MiB'>16384</memory>
  <vcpu>8</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <q35/>
  <cpu mode='host-passthrough'/>
  <interface type='network'>
    <source network='default'/>
    <model type='virtio'/>
  </interface>
  <disk type='file' device='disk'>
    <driver name='qemu' type='qcow2'/>
    <source file='/var/lib/libvirt/images/vimi-c2.qcow2'/>
    <target dev='vda' bus='virtio'/>
  </disk>
</domain>`

const VIMIAlertVMXML = `<domain type='kvm'>
  <name>vimi-alert-processor</name>
  <memory unit='MiB'>8192</memory>
  <vcpu>4</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <q35/>
  <cpu mode='host-passthrough'/>
  <interface type='network'>
    <source network='default'/>
    <model type='virtio'/>
  </interface>
  <disk type='file' device='disk'>
    <driver name='qemu' type='qcow2'/>
    <source file='/var/lib/libvirt/images/vimi-alert.qcow2'/>
    <target dev='vda' bus='virtio'/>
  </disk>
</domain>`

const VIMIReplayVMXML = `<domain type='kvm">
  <name>vimi-replay-server</name>
  <memory unit='MiB'>32768</memory>
  <vcpu>16</vcpu>
  <os>
    <type arch='x86_64'>hvm</type>
    <boot dev='hd'/>
  </os>
  <q35/>
  <cpu mode='host-passthrough'/>
  <interface type='network'>
    <source network='default'/>
    <model type='virtio'/>
  </interface>
  <disk type='file' device='disk'>
    <driver name='qemu' type='qcow2'/>
    <source file='/var/lib/libvirt/images/vimi-replay.qcow2'/>
    <target dev='vda' bus='virtio'/>
  </disk>
</domain>`

// XML helpers
func init() {
    // Register XML template types
    _ = xml.RawXML()
}

// ValidateMulticast checks if a multicast address is valid for DIS
func ValidateMulticast(addr string) error {
    u, err := url.Parse("udp://" + addr)
    if err != nil {
        return fmt.Errorf("invalid multicast address: %w", err)
    }
    host := u.Hostname()
    // DIS uses 239.255.0.0 - 239.255.255.255
    if host < "239.255.0.0" || host > "239.255.255.255" {
        return fmt.Errorf("DIS multicast must be 239.255.0.0 - 239.255.255.255")
    }
    return nil
}
