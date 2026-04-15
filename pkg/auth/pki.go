// Package auth provides PKI management
package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// PKIManager manages PKI operations
type PKIManager struct {
	caCert   *x509.Certificate
	caKey    interface{}
	certPool *x509.CertPool
	crl      *x509.RevocationList
	revoked  map[string]bool
}

// PKIConfig holds PKI configuration
type PKIConfig struct {
	CACertFile   string
	CAKeyFile    string
	KeySize      int
	ValidityDays int
}

// DefaultPKIConfig returns default PKI configuration
func DefaultPKIConfig() *PKIConfig {
	return &PKIConfig{
		KeySize:      4096,
		ValidityDays: 365,
	}
}

// NewPKIManager creates a new PKI manager
func NewPKIManager(config *PKIConfig) (*PKIManager, error) {
	pki := &PKIManager{
		revoked:  make(map[string]bool),
		certPool: x509.NewCertPool(),
	}

	// Load or generate CA
	if config.CACertFile != "" {
		if err := pki.loadCA(config.CACertFile, config.CAKeyFile); err != nil {
			return nil, fmt.Errorf("failed to load CA: %w", err)
		}
	} else {
		if err := pki.generateCA(config.KeySize, config.ValidityDays); err != nil {
			return nil, fmt.Errorf("failed to generate CA: %w", err)
		}
	}

	return pki, nil
}

// loadCA loads an existing CA certificate and key
func (p *PKIManager) loadCA(certFile, keyFile string) error {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read CA cert: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA cert PEM")
	}

	p.caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	p.caKey, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		p.caKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse CA key: %w", err)
		}
	}

	p.certPool.AddCert(p.caCert)
	return nil
}

// generateCA generates a new CA certificate and key
func (p *PKIManager) generateCA(keySize, validityDays int) error {
	// Generate RSA key
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %w", err)
	}

	// Generate serial number
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial: %w", err)
	}

	// Create CA certificate template
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"VIGIL PKI"},
			CommonName:   "VIGIL Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, validityDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign CA certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create CA cert: %w", err)
	}

	p.caCert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA cert: %w", err)
	}

	p.caKey = key
	p.certPool.AddCert(p.caCert)

	return nil
}

// CertificateRequest holds certificate request parameters
type CertificateRequest struct {
	CommonName   string
	Organization string
	DNSNames     []string
	IPAddresses  []string
	KeyType      string // "RSA" or "ECDSA"
	KeySize      int
	Days         int
	IsClient     bool
}

// GenerateCertificate generates a new certificate signed by CA
func (p *PKIManager) GenerateCertificate(req *CertificateRequest) ([]byte, []byte, error) {
	// Generate key
	var key interface{}
	var publicKey interface{}
	var err error

	switch req.KeyType {
	case "ECDSA":
		var curve elliptic.Curve
		switch req.KeySize {
		case 224:
			curve = elliptic.P224()
		case 256:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			curve = elliptic.P256()
		}
		key, err = ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}
		publicKey = &(key.(*ecdsa.PrivateKey)).PublicKey
	default: // RSA
		size := req.KeySize
		if size == 0 {
			size = 2048
		}
		key, err = rsa.GenerateKey(rand.Reader, size)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}
		publicKey = &(key.(*rsa.PrivateKey)).PublicKey
	}

	// Generate serial number
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{req.Organization},
			CommonName:   req.CommonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, req.Days),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if req.IsClient {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	// Add DNS names
	template.DNSNames = req.DNSNames

	// Add IP addresses
	for _, ip := range req.IPAddresses {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(ip))
	}

	// Sign certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, p.caCert, publicKey, p.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	return certPEM, keyPEM, nil
}

// RevokeCertificate revokes a certificate
func (p *PKIManager) RevokeCertificate(serialNumber string) error {
	p.revoked[serialNumber] = true
	return p.updateCRL()
}

// IsRevoked checks if a certificate is revoked
func (p *PKIManager) IsRevoked(serialNumber string) bool {
	return p.revoked[serialNumber]
}

// updateCRL updates the certificate revocation list
func (p *PKIManager) updateCRL() error {
	// In production, this would generate a proper CRL
	// using x509.CreateRevocationList
	return nil
}

// GetCACertificate returns the CA certificate in PEM format
func (p *PKIManager) GetCACertificate() ([]byte, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: p.caCert.Raw,
	})
	return certPEM, nil
}

// GetCAKey returns the CA private key in PEM format (for backup only)
func (p *PKIManager) GetCAKey() ([]byte, error) {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(p.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CA key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})
	return keyPEM, nil
}

// VerifyCertificate verifies a certificate against CA
func (p *PKIManager) VerifyCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if revoked
	if p.IsRevoked(cert.SerialNumber.String()) {
		return nil, fmt.Errorf("certificate is revoked")
	}

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots: p.certPool,
	}

	_, err = cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	return cert, nil
}

// CertValidity holds certificate validity information
type CertValidity struct {
	NotBefore time.Time
	NotAfter  time.Time
	ExpiresIn time.Duration
	IsExpired bool
	IsRevoked bool
}

// CheckCertValidity checks certificate validity
func (p *PKIManager) CheckCertValidity(certPEM []byte) (*CertValidity, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	now := time.Now()
	validity := &CertValidity{
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		ExpiresIn: time.Until(cert.NotAfter),
		IsExpired: now.After(cert.NotAfter),
		IsRevoked: p.IsRevoked(cert.SerialNumber.String()),
	}

	return validity, nil
}

// SaveCertificate saves certificate to file
func SaveCertificate(filename string, certPEM []byte) error {
	return os.WriteFile(filename, certPEM, 0644)
}

// SaveKey saves private key to file
func SaveKey(filename string, keyPEM []byte) error {
	return os.WriteFile(filename, keyPEM, 0600)
}

// LoadCertificate loads certificate from file
func LoadCertificate(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// LoadKey loads private key from file
func LoadKey(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
