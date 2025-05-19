package main

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	p11 "github.com/ThalesGroup/crypto11"
)

func main() {
	// Load certificate
	cert, err := loadCertificate("client_cert.pem")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	userPIN := os.Getenv("HSM_USER_PIN")
	if userPIN == "" {
		log.Fatal("HSM_USER_PIN environment variable is not set")
	}
	keyLabel := os.Getenv("HSM_KEY_LABEL")
	if keyLabel == "" {
		log.Fatal("HSM_KEY_LABEL environment variable is not set")
	}

	// Configure PKCS#11
	ctx, err := p11.Configure(&p11.Config{
		Path:       "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so", // CloudHSM PKCS#11 library
		TokenLabel: "CLOUDHSM",                                // or use TokenSerial
		Pin:        userPin,                                   // typically CU or CO user
	})
	if err != nil {
		log.Fatalf("Failed to initialize crypto11: %v", err)
	}
	defer ctx.Close()

	// Find key by label
	signer, err := ctx.FindKeyPair(nil, []byte(keyLabel))
	if err != nil {
		log.Fatalf("Failed to find key: %v", err)
	}
	if signer == nil {
		log.Fatalf("No signer found with label: %s", keyLabel)
	}

	// Wrap certificate and signer
	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  signer.(crypto.Signer),
		Leaf:        cert,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Perform request
	resp, err := client.Get("https://honorato.org/hsm")
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", body)
}

// loadCertificate loads a PEM-encoded x509 certificate from file
func loadCertificate(certPath string) (*x509.Certificate, error) {
	pemData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode PEM block containing certificate")
	}

	return x509.ParseCertificate(block.Bytes)
}
