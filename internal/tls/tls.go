package tlsutil

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
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// GenerateSelfSignedCert generates a self-signed certificate and returns a tls.Certificate
func GenerateSelfSignedCert() (tls.Certificate, error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"HCP Checkmarx Listener"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost", "hcp-checkmarx-listener"},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %v", err)
	}

	// Create tls.Certificate from DER bytes and private key
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	log.Info("Generated self-signed TLS certificate for HCP Checkmarx Listener")
	log.Info("Certificate valid for: localhost, 127.0.0.1, ::1, hcp-checkmarx-listener")
	log.Info("Certificate expires: " + template.NotAfter.Format(time.RFC3339))

	return cert, nil
}

// WriteCertificateFiles writes the certificate and key to files for persistence
func WriteCertificateFiles(cert tls.Certificate, certFile, keyFile string) error {
	// Write certificate file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Write private key file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer keyOut.Close()

	privateKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not RSA")
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	log.Infof("Wrote TLS certificate to: %s", certFile)
	log.Infof("Wrote TLS private key to: %s", keyFile)

	return nil
}

// LoadTLSConfig loads TLS configuration based on the provided parameters
func LoadTLSConfig(tlsEnabled bool, certFile, keyFile string, selfSigned bool) (*tls.Config, error) {
	if !tlsEnabled {
		return nil, nil
	}

	var cert tls.Certificate
	var err error

	// Check if certificate files exist and are provided
	if certFile != "" && keyFile != "" {
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			if selfSigned {
				log.Warnf("Certificate file %s does not exist, generating self-signed certificate", certFile)
			} else {
				return nil, fmt.Errorf("certificate file %s does not exist", certFile)
			}
		} else if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			if selfSigned {
				log.Warnf("Key file %s does not exist, generating self-signed certificate", keyFile)
			} else {
				return nil, fmt.Errorf("key file %s does not exist", keyFile)
			}
		} else {
			// Load existing certificate files
			cert, err = tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				if selfSigned {
					log.Warnf("Failed to load certificate files: %v, generating self-signed certificate", err)
				} else {
					return nil, fmt.Errorf("failed to load certificate files: %v", err)
				}
			} else {
				log.Infof("Loaded TLS certificate from: %s", certFile)
				log.Infof("Loaded TLS private key from: %s", keyFile)
				return &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS12,
				}, nil
			}
		}
	}

	// Generate self-signed certificate if needed
	if selfSigned || (certFile != "" && keyFile != "") {
		cert, err = GenerateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %v", err)
		}

		// If file paths are provided, save the certificate
		if certFile != "" && keyFile != "" {
			if err := WriteCertificateFiles(cert, certFile, keyFile); err != nil {
				log.Warnf("Failed to write certificate files: %v", err)
			}
		}

		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, nil
	}

	return nil, fmt.Errorf("TLS enabled but no certificate configuration provided")
}
