package c2bmc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPKIAuth tests PKI authenticator creation
func TestPKIAuth(t *testing.T) {
	// Skip - requires valid certificates
	t.Skip("requires valid test certificates")
}

// TestPKIAuthWithCerts tests PKI auth with generated certs
func TestPKIAuthWithCerts(t *testing.T) {
	// Generate temporary certs
	tmpDir, err := os.MkdirTemp("", "pki-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificate
	certFile, keyFile, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	// Test creating PKI auth
	auth, err := NewPKIAuth(&PKIConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		ServerName:         "test.example.mil",
		InsecureSkipVerify: false,
	})
	if err != nil {
		t.Fatalf("NewPKIAuth() error = %v", err)
	}

	if auth == nil {
		t.Fatal("NewPKIAuth() returned nil")
	}
}

// TestPKIAuthNoConfig tests PKI auth without config
func TestPKIAuthNoConfig(t *testing.T) {
	_, err := NewPKIAuth(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
}

// TestPKIAuthNoCert tests PKI auth without certificate
func TestPKIAuthNoCert(t *testing.T) {
	auth, err := NewPKIAuth(&PKIConfig{
		CertFile: "",
		KeyFile:  "",
	})
	if err != nil {
		t.Fatalf("NewPKIAuth() error = %v", err)
	}

	// Validate should fail without cert
	if err := auth.Validate(); err == nil {
		t.Error("Expected validation error for missing cert")
	}
}

// TestTLSConfig tests TLS configuration generation
func TestTLSConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile, keyFile, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	auth, err := NewPKIAuth(&PKIConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		ServerName:         "test.example.mil",
		InsecureSkipVerify: false,
	})
	if err != nil {
		t.Fatalf("NewPKIAuth() error = %v", err)
	}

	tlsConfig := auth.TLSConfig()
	if tlsConfig == nil {
		t.Fatal("TLSConfig() returned nil")
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(tlsConfig.Certificates))
	}

	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected TLS 1.2, got %d", tlsConfig.MinVersion)
	}

	if len(tlsConfig.CipherSuites) == 0 {
		t.Error("Expected cipher suites")
	}
}

// TestMTLSConfig tests mutual TLS configuration
func TestMTLSConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile, keyFile, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	tlsConfig, err := MTLSConfig(certFile, keyFile, caFile, "test.example.mil")
	if err != nil {
		t.Fatalf("MTLSConfig() error = %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("MTLSConfig() returned nil")
	}

	if tlsConfig.ServerName != "test.example.mil" {
		t.Errorf("Expected ServerName 'test.example.mil', got %s", tlsConfig.ServerName)
	}
}

// TestLoadCertificate tests certificate loading
func TestLoadCertificate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile, keyFile, _, err := generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	cert, err := LoadCertificate(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadCertificate() error = %v", err)
	}

	if cert.PrivateKey == nil {
		t.Error("Expected private key")
	}
}

// TestLoadCA tests CA certificate loading
func TestLoadCA(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pki-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, _, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test certs: %v", err)
	}

	caPool, err := LoadCA(caFile)
	if err != nil {
		t.Fatalf("LoadCA() error = %v", err)
	}

	if caPool == nil {
		t.Error("LoadCA() returned nil")
	}
}

// TestLoadCAInvalid tests CA loading with invalid file
func TestLoadCAInvalid(t *testing.T) {
	_, err := LoadCA("/nonexistent/ca.crt")
	if err == nil {
		t.Error("Expected error for nonexistent CA file")
	}
}

// generateTestCerts generates test certificates
func generateTestCerts(tmpDir string) (certFile, keyFile, caFile string, err error) {
	// Generate CA key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}

	// Generate CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return "", "", "", err
	}

	caFile = filepath.Join(tmpDir, "ca.crt")
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}), 0644); err != nil {
		return "", "", "", err
	}

	// Generate client key
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", err
	}

	// Generate client certificate
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
			CommonName:   "test.example.mil",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		return "", "", "", err
	}

	certFile = filepath.Join(tmpDir, "client.crt")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertBytes}), 0644); err != nil {
		return "", "", "", err
	}

	keyFile = filepath.Join(tmpDir, "client.key")
	keyBytes := x509.MarshalPKCS1PrivateKey(clientKey)
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}), 0600); err != nil {
		return "", "", "", err
	}

	return certFile, keyFile, caFile, nil
}

// BenchmarkPKIAuthCreation benchmarks PKI auth creation
func BenchmarkPKIAuthCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "pki-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile, keyFile, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		b.Fatalf("Failed to generate test certs: %v", err)
	}

	config := &PKIConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		ServerName:         "test.example.mil",
		InsecureSkipVerify: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPKIAuth(config)
	}
}

// BenchmarkTLSConfigCreation benchmarks TLS config creation
func BenchmarkTLSConfigCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "pki-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	certFile, keyFile, caFile, err := generateTestCerts(tmpDir)
	if err != nil {
		b.Fatalf("Failed to generate test certs: %v", err)
	}

	auth, err := NewPKIAuth(&PKIConfig{
		CertFile:           certFile,
		KeyFile:            keyFile,
		CAFile:             caFile,
		ServerName:         "test.example.mil",
		InsecureSkipVerify: false,
	})
	if err != nil {
		b.Fatalf("Failed to create auth: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.TLSConfig()
	}
}
