package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	targetURL      string
	listenAddr     = "127.0.0.1:8443" // Changed default to 8443 for HTTPS
	requestTimeout = 15 * time.Second
	maxRetries     = 2
	insecureTLS    = false // This is for target Https

	certFile = "/etc/resilient-proxy/proxy.crt"
	keyFile  = "/etc/resilient-proxy/proxy.key"
)

func main() {
	// Required: Target
	targetURL = os.Getenv("PI_TARGET")
	if targetURL == "" {
		log.Println("❌ ERROR: PI_TARGET environment variable is required but not set.")
		log.Println("   Example: PI_TARGET=https://service.example.com")
		log.Println("   Or in systemd: Environment=PI_TARGET=https://...")
		os.Exit(1) // Explicit exit
	}

	// Optional settings
	if addr := os.Getenv("PI_LISTEN"); addr != "" {
		listenAddr = addr
	}
	if v := os.Getenv("PI_INSECURE"); v != "" {
		insecureTLS, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("PI_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			requestTimeout = d
		}
	}
	if v := os.Getenv("PI_RETRIES"); v != "" {
		if r, err := strconv.Atoi(v); err == nil && r >= 0 {
			maxRetries = r
		}
	}
	if c := os.Getenv("PI_CERT"); c != "" {
		certFile = c
	}
	if k := os.Getenv("PI_KEY"); k != "" {
		keyFile = k
	}

	// Generate self-signed certificate if not exists
	if err := ensureCertificate(); err != nil {
		log.Fatalf("❌ Failed to generate certificate: %v", err)
	}

	http.HandleFunc("/", handler)

	log.Printf("✅ Secure Proxy started")
	log.Printf("   Listening on : https://%s", listenAddr)
	log.Printf("   Forwarding to : %s", targetURL)
	if insecureTLS {
		log.Printf("⚠️  Insecure TLS to target ENABLED")
	}

	log.Fatal(http.ListenAndServeTLS(listenAddr, certFile, keyFile, nil))
}

// Generate self-signed cert if not present
func ensureCertificate() error {
	if _, err := os.Stat(certFile); err == nil {
		return nil // Cert already exists
	}

	log.Println("🔑 Generating new self-signed certificate...")

	// Create directory
	os.MkdirAll("/etc/resilient-proxy", 0755)

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(5 * 365 * 24 * time.Hour) // 5 years

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"resilient-proxy Local Proxy"},
			CommonName:   "localhost",
		},
		DNSNames:              []string{"localhost", "127.0.0.1"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write cert
	certOut, _ := os.Create(certFile)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	// Write key
	keyOut, _ := os.Create(keyFile)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()

	os.Chmod(certFile, 0644)
	os.Chmod(keyFile, 0600)

	log.Printf("✅ Self-signed certificate generated: %s", certFile)
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DisableKeepAlives:     true,
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 9 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureTLS,
			},
		},
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

		target := targetURL + r.URL.RequestURI()
		req, _ := http.NewRequestWithContext(ctx, r.Method, target, bytes.NewReader(bodyBytes))

		for k, vv := range r.Header {
			for _, v := range vv {
				req.Header.Add(k, v)
			}
		}

		resp, err := client.Do(req)
		cancel()

		if err == nil {
			defer resp.Body.Close()
			for k, vv := range resp.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)

			log.Printf("OK %s %s (%dms)", r.Method, r.URL.Path, time.Since(start).Milliseconds())
			return
		}

		lastErr = err
		log.Printf("Attempt %d failed: %v", attempt+1, err)

		if attempt < maxRetries {
			time.Sleep(400 * time.Millisecond)
		}
	}

	log.Printf("❌ Proxy failed: %v", lastErr)
	http.Error(w, "Target unavailable", http.StatusGatewayTimeout)
}
