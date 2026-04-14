package c2bmc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// PKIConfig holds PKI authentication configuration
type PKIConfig struct {
	// Client certificate and key
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`

	// CA certificate for server verification
	CAFile string `json:"ca_file"`

	// Server name for certificate verification
	ServerName string `json:"server_name"`

	// Skip server certificate verification (insecure)
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

// PKIAuth handles PKI authentication for C2BMC
type PKIAuth struct {
	config *PKIConfig
	cert   tls.Certificate
	caPool *x509.CertPool
}

// NewPKIAuth creates a new PKI authenticator
func NewPKIAuth(config *PKIConfig) (*PKIAuth, error) {
	if config == nil {
		return nil, fmt.Errorf("PKI config is required")
	}

	auth := &PKIAuth{
		config: config,
	}

	// Load client certificate
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		auth.cert = cert
	}

	// Load CA certificate
	if config.CAFile != "" {
		caPool := x509.NewCertPool()
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		auth.caPool = caPool
	}

	return auth, nil
}

// TLSConfig creates a TLS configuration for HTTP client
func (a *PKIAuth) TLSConfig() *tls.Config {
	config := &tls.Config{
		InsecureSkipVerify: a.config.InsecureSkipVerify,
	}

	// Add client certificate
	if a.cert.PrivateKey != nil {
		config.Certificates = []tls.Certificate{a.cert}
	}

	// Add CA for server verification
	if a.caPool != nil {
		config.RootCAs = a.caPool
	}

	// Set server name for verification
	if a.config.ServerName != "" {
		config.ServerName = a.config.ServerName
	}

	// Minimum TLS version
	config.MinVersion = tls.VersionTLS12

	// Preferred cipher suites
	config.CipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	return config
}

// Validate validates the PKI configuration
func (a *PKIAuth) Validate() error {
	if a.config.CertFile == "" {
		return fmt.Errorf("client certificate file is required")
	}
	if a.config.KeyFile == "" {
		return fmt.Errorf("client key file is required")
	}
	if a.cert.PrivateKey == nil {
		return fmt.Errorf("failed to load client certificate")
	}
	return nil
}

// CertPool returns the CA certificate pool
func (a *PKIAuth) CertPool() *x509.CertPool {
	return a.caPool
}

// CertificateInfo represents certificate information
type CertificateInfo struct {
	Subject      string `json:"subject"`
	Issuer       string `json:"issuer"`
	SerialNumber string `json:"serial_number"`
	NotBefore    string `json:"not_before"`
	NotAfter     string `json:"not_after"`
	IsExpired    bool   `json:"is_expired"`
	IsValid      bool   `json:"is_valid"`
}

// GetCertificateInfo returns information about the loaded certificate
func (a *PKIAuth) GetCertificateInfo() (*CertificateInfo, error) {
	if a.cert.Leaf == nil {
		// Parse certificate if not already parsed
		var err error
		a.cert.Leaf, err = x509.ParseCertificate(a.cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
	}

	cert := a.cert.Leaf
	now := timeNow()

	return &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore.Format("2006-01-02 15:04:05 UTC"),
		NotAfter:     cert.NotAfter.Format("2006-01-02 15:04:05 UTC"),
		IsExpired:    now.After(cert.NotAfter),
		IsValid:      now.After(cert.NotBefore) && now.Before(cert.NotAfter),
	}, nil
}

// timeNow is a variable for testing
var timeNow = func() time.Time { return time.Now() }

// VerifyCertificate verifies a certificate against the CA
func (a *PKIAuth) VerifyCertificate(certPEM []byte) error {
	if a.caPool == nil {
		return fmt.Errorf("no CA certificate loaded")
	}

	cert, err := x509.ParseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	opts := x509.VerifyOptions{
		Roots: a.caPool,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// MTLSConfig creates mutual TLS configuration
func MTLSConfig(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	auth, err := NewPKIAuth(&PKIConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		ServerName:         serverName,
		InsecureSkipVerify: false,
	})
	if err != nil {
		return nil, err
	}

	if err := auth.Validate(); err != nil {
		return nil, err
	}

	return auth.TLSConfig(), nil
}

// LoadCertificate loads a certificate from files
func LoadCertificate(certFile, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

// LoadCA loads a CA certificate from a file
func LoadCA(caFile string) (*x509.CertPool, error) {
	caPool := x509.NewCertPool()
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return caPool, nil
}