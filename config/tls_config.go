package config

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"golang.org/x/crypto/acme/autocert"
	"henrymail/database"
	"henrymail/models"
	"log"
	"math"
	"math/big"
	"net"
	"strings"
	"time"
)

func GetTLSConfig() *tls.Config {
	// Don't try to use TLS if nothing requires it
	if !GetBool(WebAdminUseTls) &&
		!GetBool(MsaUseTls) &&
		!GetBool(MtaUseTls) &&
		!GetBool(ImapUseTls) {
		log.Println("Not using TLS for any services")
		return nil
	}

	certMode := CertMode(GetString(CertificateMode))
	switch certMode {
	case AutoCert:
		m := &autocert.Manager{
			Cache:      &dbCache{},
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(GetString(ServerName)),
			Email:      GetString(AutoCertEmail),
		}
		return m.TLSConfig()
	case SelfSigned:
		c, e := GenerateCert(GetString(ServerName), time.Duration(math.MaxInt64), false, 1024)
		if e != nil {
			log.Fatal(e)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
		}
	case Given:
		c, e := tls.LoadX509KeyPair(GetString(CertificateFile), GetString(KeyFile))
		if e != nil {
			log.Fatal(e)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
		}
	default:
		log.Fatalf("Unexpected certificate mode %v", certMode)
	}
	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			log.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

/**
host       = flag.String("host", "", "Comma-separated hostnames and IPs to generate a certificate for")
validFor   = flag.Duration("duration", 365*24*time.Hour, "Duration that certificate is valid for")
isCA       = flag.Bool("ca", false, "whether this cert should be its own Certificate Authority")
rsaBits    = flag.Int("rsa-bits", 2048, "Size of RSA key to generate. Ignored if --ecdsa-curve is set")
*/
func GenerateCert(host string,
	validFor time.Duration,
	isCA bool,
	rsaBits int) (tls.Certificate, error) {
	if len(host) == 0 {
		log.Fatalf("Missing required --host parameter")
	}

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)

	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut := &bytes.Buffer{}
	keyOut := &bytes.Buffer{}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("failed to write data to cert.pem: %s", err)
	}
	if err := pem.Encode(keyOut, pemBlockForKey(priv)); err != nil {
		log.Fatalf("failed to write data to key.pem: %s", err)
	}
	return tls.X509KeyPair(certOut.Bytes(), keyOut.Bytes())
}

type dbCache struct{}

func (dbCache) Get(ctx context.Context, key string) ([]byte, error) {
	dbKey, e := models.KeyByName(database.DB, dbKeyName(key))
	if e != nil {
		return nil, e
	}
	return dbKey.Key, nil
}

func (dbCache) Put(ctx context.Context, key string, data []byte) error {
	dbKey, e := models.KeyByName(database.DB, dbKeyName(key))
	if e != nil {
		dbKey = &models.Key{
			Name: dbKeyName(key),
		}
	}
	dbKey.Key = data
	return dbKey.Save(database.DB)
}

func (dbCache) Delete(ctx context.Context, key string) error {
	dbKey, e := models.KeyByName(database.DB, dbKeyName(key))
	if e != nil {
		return e
	}
	return dbKey.Delete(database.DB)
}

func dbKeyName(key string) string {
	return "autocert-" + key
}
