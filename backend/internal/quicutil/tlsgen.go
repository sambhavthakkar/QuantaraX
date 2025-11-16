// Package quicutil provides QUIC utilities and TLS helpers for QuantaraX.
package quicutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// GenerateSelfSignedCert generates a self-signed TLS certificate for development use.
//
// This function creates an RSA certificate valid for 365 days.
// The certificate is suitable for local development and testing but should
// NOT be used in production.
//
// Returns:
//   - certPEM: PEM-encoded certificate
//   - keyPEM: PEM-encoded private key
//   - error: Non-nil if generation fails
//
// Security Warning:
//   - Self-signed certificates should only be used with InsecureSkipVerify=true
//   - Production deployments must use proper certificate validation
func GenerateSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	// Generate RSA private key (2048 bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Set up certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"QuantaraX Development"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return certPEM, keyPEM, nil
}

// MakeTLSConfig creates a tls.Config from PEM-encoded certificate and key.
//
// Parameters:
//   - certPEM: PEM-encoded certificate
//   - keyPEM: PEM-encoded private key
//
// Returns:
//   - *tls.Config configured for TLS 1.3 with the provided certificate
//   - error if loading the certificate fails
func MakeTLSConfig(certPEM, keyPEM []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to load key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13, // Enforce TLS 1.3
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

// MakeClientTLSConfig creates a client tls.Config for development use.
//
// This configuration skips certificate verification (InsecureSkipVerify=true)
// and should ONLY be used for local development and testing.
//
// Returns:
//   - *tls.Config configured for insecure client connections
func MakeClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // WARNING: Only for development
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
	}
}