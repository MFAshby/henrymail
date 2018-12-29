package main

import (
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"
	"log"
)

func GetTLSConfig() *tls.Config {
	if GetBool(UseAutoCertKey) {
		m := &autocert.Manager{
			Cache:      autocert.DirCache("keys"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(GetString(DomainKey)),
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
