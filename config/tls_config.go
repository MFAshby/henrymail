package config

import (
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"
	"log"
)

func GetTLSConfig() *tls.Config {
	// Don't try to use TLS if nothing requires it
	if !GetBool(WebAdminUseTls) &&
		!GetBool(MsaUseTls) &&
		!GetBool(MtaUseTls) &&
		!GetBool(ImapUseTls) {
		return nil
	}

	if GetBool(UseAutoCert) {
		m := &autocert.Manager{
			Cache:      autocert.DirCache("keys"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(GetString(ServerName)),
			Email:      GetString(AutoCertEmail),
		}
		return m.TLSConfig()
	} else {
		c, e := tls.LoadX509KeyPair(GetString(CertificateFile), GetString(KeyFile))
		if e != nil {
			log.Fatal(e)
		}
		return &tls.Config{
			Certificates: []tls.Certificate{c},
		}
	}
}
