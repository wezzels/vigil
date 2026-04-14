// Package auth provides authentication and authorization for VIGIL
package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// mTLSConfig holds mTLS configuration
type mTLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string
	MinVersion uint16
}

// DefaultmTLSConfig returns default mTLS configuration
func DefaultmTLSConfig() *mTLSConfig {
	return &mTLSConfig{
		MinVersion: tls.VersionTLS12,
	}
}

// MTLSManager manages mTLS connections
type MTLSManager struct {
	config     *mTLSConfig
	certPool   *x509.CertPool
	certificate tls.Certificate
}

// NewMTLSManager creates a new mTLS manager
func NewMTLSManager(config *mTLSConfig) (*MTLSManager, error) {
	mgr := &MTLSManager{
		config: config,
	}

	// Load CA certificate
	if err := mgr.loadCA(); err != nil {
		return nil, fmt.Errorf("failed to load CA: %w", err)
	}

	// Load server/client certificate
	if err := mgr.loadCertificate(); err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	return mgr, nil
}

// loadCA loads the CA certificate
func (m *MTLSManager) loadCA() error {
	caCert, err := os.ReadFile(m.config.CAFile)
	if err != nil {
		return fmt.Errorf("failed to read CA file: %w", err)
	}

	m.certPool = x509.NewCertPool()
	if !m.certPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to append CA certificate")
	}

	return nil
}

// loadCertificate loads the server/client certificate
func (m *MTLSManager) loadCertificate() error {
	cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}
	m.certificate = cert
	return nil
}

// ServerTLSConfig returns TLS configuration for server
func (m *MTLSManager) ServerTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{m.certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    m.certPool,
		MinVersion:   m.config.MinVersion,
		ServerName:   m.config.ServerName,
	}
}

// ClientTLSConfig returns TLS configuration for client
func (m *MTLSManager) ClientTLSConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{m.certificate},
		RootCAs:      m.certPool,
		MinVersion:   m.config.MinVersion,
		ServerName:   m.config.ServerName,
	}
}

// VerifyPeer verifies a peer certificate
func (m *MTLSManager) VerifyPeer(rawCerts [][]byte) error {
	certs := make([]*x509.Certificate, len(rawCerts))
	for i, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}
		certs[i] = cert
	}

	opts := x509.VerifyOptions{
		Roots:         m.certPool,
		Intermediates: x509.NewCertPool(),
	}

	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}

	_, err := certs[0].Verify(opts)
	return err
}

// GetCertificate returns the current certificate
func (m *MTLSManager) GetCertificate() *tls.Certificate {
	return &m.certificate
}

// ReloadCertificate reloads the certificate from disk
func (m *MTLSManager) ReloadCertificate() error {
	return m.loadCertificate()
}

// CertificateInfo contains certificate information
type CertificateInfo struct {
	Subject      string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	SerialNumber string
	DNSNames     []string
	IPAddresses  []net.IP
}

// GetCertificateInfo returns information about a certificate
func GetCertificateInfo(cert *tls.Certificate) (*CertificateInfo, error) {
	if cert == nil || len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &CertificateInfo{
		Subject:      x509Cert.Subject.String(),
		Issuer:       x509Cert.Issuer.String(),
		NotBefore:    x509Cert.NotBefore,
		NotAfter:     x509Cert.NotAfter,
		SerialNumber: x509Cert.SerialNumber.String(),
		DNSNames:     x509Cert.DNSNames,
		IPAddresses:  x509Cert.IPAddresses,
	}, nil
}

// IsExpired checks if certificate is expired
func (info *CertificateInfo) IsExpired() bool {
	return time.Now().After(info.NotAfter)
}

// ExpiresSoon checks if certificate expires within duration
func (info *CertificateInfo) ExpiresSoon(d time.Duration) bool {
	return time.Until(info.NotAfter) < d
}

// TLSListener wraps net.Listener with TLS
type TLSListener struct {
	net.Listener
	config *tls.Config
}

// NewTLSListener creates a new TLS listener
func NewTLSListener(listener net.Listener, config *tls.Config) *TLSListener {
	return &TLSListener{
		Listener: listener,
		config:   config,
	}
}

// Accept accepts a new TLS connection
func (l *TLSListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Server(conn, l.config)
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	return tlsConn, nil
}

// TLSConn wraps tls.Conn with additional functionality
type TLSConn struct {
	*tls.Conn
	peerCerts []*x509.Certificate
}

// NewTLSConn creates a new TLS connection wrapper
func NewTLSConn(conn *tls.Conn) (*TLSConn, error) {
	state := conn.ConnectionState()
	return &TLSConn{
		Conn:      conn,
		peerCerts: state.PeerCertificates,
	}, nil
}

// GetPeerCertificate returns the peer certificate
func (c *TLSConn) GetPeerCertificate() *x509.Certificate {
	if len(c.peerCerts) == 0 {
		return nil
	}
	return c.peerCerts[0]
}

// GetPeerCertificates returns all peer certificates
func (c *TLSConn) GetPeerCertificates() []*x509.Certificate {
	return c.peerCerts
}

// VerifyPeerCertificate verifies peer certificate manually
func (c *TLSConn) VerifyPeerCertificate(opts x509.VerifyOptions) error {
	if len(c.peerCerts) == 0 {
		return fmt.Errorf("no peer certificates")
	}

	opts.Intermediates = x509.NewCertPool()
	for _, cert := range c.peerCerts[1:] {
		opts.Intermediates.AddCert(cert)
	}

	_, err := c.peerCerts[0].Verify(opts)
	return err
}

// TLSDialer provides TLS dial functionality
type TLSDialer struct {
	config *tls.Config
	timeout time.Duration
}

// NewTLSDialer creates a new TLS dialer
func NewTLSDialer(config *tls.Config, timeout time.Duration) *TLSDialer {
	return &TLSDialer{
		config: config,
		timeout: timeout,
	}
}

// Dial connects to a TLS server
func (d *TLSDialer) Dial(ctx context.Context, network, addr string) (*TLSConn, error) {
	dialer := &tls.Dialer{
		Config: d.config,
	}

	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("not a TLS connection")
	}

	// Get TLS connection wrapper
	return NewTLSConn(tlsConn)
}

// DialWithTimeout dials with a timeout
func (d *TLSDialer) DialWithTimeout(network, addr string) (*TLSConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	return d.Dial(ctx, network, addr)
}

// LoadCertificates loads multiple certificates from files
func LoadCertificates(certFiles, keyFiles []string) ([]tls.Certificate, error) {
	if len(certFiles) != len(keyFiles) {
		return nil, fmt.Errorf("mismatched cert and key files")
	}

	certs := make([]tls.Certificate, len(certFiles))
	for i := range certFiles {
		cert, err := tls.LoadX509KeyPair(certFiles[i], keyFiles[i])
		if err != nil {
			return nil, fmt.Errorf("failed to load cert %d: %w", i, err)
		}
		certs[i] = cert
	}

	return certs, nil
}

// GenerateCertRequest generates a certificate request (CSR) - placeholder
// In production, use crypto/x509 to generate actual CSR
func GenerateCertRequest(commonName string, dnsNames []string) ([]byte, error) {
	// This is a placeholder - real implementation would use:
	// - x509.CreateCertificateRequest
	// - rsa.GenerateKey or ecdsa.GenerateKey
	return []byte(fmt.Sprintf("CSR for %s with DNS names %v", commonName, dnsNames)), nil
}

// CertRotationManager manages certificate rotation
type CertRotationManager struct {
	mgr        *MTLSManager
	certFile   string
	keyFile    string
	interval   time.Duration
	stopChan   chan struct{}
	reloadChan chan struct{}
}

// NewCertRotationManager creates a certificate rotation manager
func NewCertRotationManager(mgr *MTLSManager, certFile, keyFile string, interval time.Duration) *CertRotationManager {
	return &CertRotationManager{
		mgr:        mgr,
		certFile:   certFile,
		keyFile:    keyFile,
		interval:   interval,
		stopChan:   make(chan struct{}),
		reloadChan: make(chan struct{}, 1),
	}
}

// Start starts the certificate rotation checker
func (c *CertRotationManager) Start() {
	go c.checkLoop()
}

// Stop stops the rotation checker
func (c *CertRotationManager) Stop() {
	close(c.stopChan)
}

// Reload triggers a certificate reload
func (c *CertRotationManager) Reload() {
	select {
	case c.reloadChan <- struct{}{}:
	default:
	}
}

// checkLoop checks for certificate rotation
func (c *CertRotationManager) checkLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mgr.ReloadCertificate()
		case <-c.reloadChan:
			c.mgr.ReloadCertificate()
		}
	}
}

// ReadAll reads all data from reader - helper function
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}