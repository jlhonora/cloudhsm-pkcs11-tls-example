package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/miekg/pkcs11"
)

type PKCS11Signer struct {
	ctx     *pkcs11.Ctx
	session pkcs11.SessionHandle
	handle  pkcs11.ObjectHandle
}

func (s *PKCS11Signer) Public() crypto.PublicKey {
	// If needed: implement logic to retrieve public key from HSM.
	// Here’s a dummy key to satisfy the interface.
	return &rsa.PublicKey{}
}

func main() {
	// Load environment variables
	userPin := os.Getenv("HSM_USER_PIN")
	keyLabel := os.Getenv("HSM_KEY_LABEL")

	if userPin == "" || keyLabel == "" {
		log.Fatal("HSM_USER_PIN and HSM_KEY_LABEL must be set")
	}

	// Load PKCS#11 module
	module := "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so"
	p := pkcs11.New(module)
	if p == nil {
		log.Fatalf("Failed to load PKCS#11 module %s", module)
	}
	err := p.Initialize()
	if err != nil {
		log.Fatalf("PKCS#11 init error: %v", err)
	}
	defer p.Destroy()
	defer p.Finalize()

	// Get first available slot
	slots, err := p.GetSlotList(true)
	if err != nil || len(slots) == 0 {
		log.Fatalf("No usable slots found: %v", err)
	}
	slot := slots[0]

	// Open session and login
	session, err := p.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		log.Fatalf("OpenSession failed: %v", err)
	}
	defer p.CloseSession(session)

	err = p.Login(session, pkcs11.CKU_USER, userPin)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	defer p.Logout(session)

	// Find the private key by label
	err = p.FindObjectsInit(session, []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
	})
	if err != nil {
		log.Fatalf("FindObjectsInit failed: %v", err)
	}

	objs, _, err := p.FindObjects(session, 1)
	if err != nil || len(objs) == 0 {
		log.Fatalf("Private key not found for label: %s", keyLabel)
	}
	privKeyHandle := objs[0]
	p.FindObjectsFinal(session)

	// Create signer
	signer := &PKCS11Signer{
		ctx:     p,
		session: session,
		handle:  privKeyHandle,
	}

	// Load certificate
	certPEM, err := os.ReadFile("client_cert.pem")
	if err != nil {
		log.Fatalf("Failed to read cert: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse cert: %v", err)
	}

	// TLS certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  signer,
		Leaf:        cert,
	}

	// TLS config
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // Set to false + RootCAs in production
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Request
	resp, err := client.Get("https://honorato.org/hsm")
	if err != nil {
		log.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response:\n%s\n", string(body))
}
