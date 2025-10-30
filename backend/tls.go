package backend

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
)

type x509Cert struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func LoadTLSConfig(hashPort int) (*tls.Config, error) {
	certPEM, privatePEM := GenerateCertDevServer(hashPort)

	cert, err := tls.X509KeyPair(certPEM, privatePEM)
	if err != nil {
		return nil, fmt.Errorf("error loading TLS certificate: %w", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3, "h2", "http/1.1"},
	}

	return tlsConfig, nil
}

// GenerateCertDevServer returns certPEM and privatePEM
func GenerateCertDevServer(hashPort int) ([]byte, []byte) {
	cert, err := genCert()
	if err != nil {
		log.Fatal(err)
	}

	go runDevHashServer(hashPort, cert)

	derBuf, _ := x509.MarshalECPrivateKey(cert.key)

	pem1 := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.cert.Raw,
	})

	private := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBuf,
	})

	return pem1, private
}

// Create a x509 Certificate
func genCert() (*x509Cert, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 10),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	ca, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &x509Cert{ca, privateKey}, nil
}

func runDevHashServer(hashPort int, cert *x509Cert) {
	hash := sha256.Sum256(cert.cert.Raw)
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", hashPort),
		Handler: mux,
	}

	log.Printf("Warning: Development hash server shoud not be on during production!\n")
	log.Printf("Starting hash server on port %d\n", hashPort)

	mux.HandleFunc("/hash", func(w http.ResponseWriter, _ *http.Request) {
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(hash[:])))
	})

	go func() {
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start dev server on port %d: %s", hashPort, err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("failed to shutdown dev hash server: %s", err)
	}
	log.Printf("Shutdown hash server\n")
}
