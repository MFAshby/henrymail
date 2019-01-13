package config

import (
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"
	"log"
)

func GetTLSConfig() *tls.Config {
	// Don't try to use TLS if nothing requires it
	if !GetBool(WebAdminUseTlsKey) &&
		!GetBool(MsaUseTlsKey) &&
		!GetBool(MtaUseTlsKey) &&
		!GetBool(ImapUseTlsKey) {
		return nil
	}

	if GetBool(UseAutoCertKey) {
		m := &autocert.Manager{
			Cache:      autocert.DirCache("keys"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(GetString(ServerNameKey)),
			Email:      GetString(AutoCertEmailKey),
		}
		return m.TLSConfig()
	} else {
		c, e := tls.LoadX509KeyPair(GetString(CertificateFileKey), GetString(KeyFileKey))
		if e != nil {
			log.Fatal(e)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
		}
	}
}
