package config

import (
	"context"
	"github.com/spf13/viper"
	"log"
	"net"
)

type CertMode string

const (
	AutoCert   CertMode = "AutoCert"
	Given               = "Given"
	SelfSigned          = "SelfSigned"
)

const (
	// The domain we're serving email for (e.g. example.com)
	Domain = "Domain"

	// Network
	// The public name of this server (e.g. mail.example.com)
	ServerName = "ServerName"

	// Ports / addresses to listen on for various services
	MsaAddress      = "MsaAddress"
	MtaAddress      = "MtaAddress"
	ImapAddress     = "ImapAddress"
	WebAdminAddress = "WebAdminAddress"

	// DNS
	DnsServer      = "DnsServer"
	FakeDns        = "FakeDns" // For testing only!
	FakeDnsAddress = "FakeDnsAddress"

	// TLS
	MtaUseTls      = "MtaUseTls"
	MsaUseTls      = "MsaUseTls"
	ImapUseTls     = "ImapUseTls"
	WebAdminUseTls = "WebAdminUseTls"

	CertificateMode = "CertificateMode"
	// If autocert is enabled
	AutoCertEmail    = "AutoCertEmail"
	AutoCertCacheDir = "AutoCertCacheDir"
	// If autocert is disabled, provide TLS certs
	CertificateFile = "CertificateFile"
	KeyFile         = "KeyFile"

	// Database
	DbDriverName       = "DbDriverName"
	DbConnectionString = "DbConnectionString"

	// Message sending stuff
	MaxIdleSeconds  = "MaxIdleSeconds"
	MaxMessageBytes = "MaxMessageBytes"
	MaxRecipients   = "MaxRecipients"
	RetryCronSpec   = "RetryCronSpec"
	RetryCount      = "RetryCount"

	// Admin stuff
	AdminUsername    = "AdminUsername"
	AdminPassword    = "AdminPassword"
	DefaultMailboxes = "DefaultMailboxes"

	// DKIM
	DkimSign           = "DkimSign"
	DkimVerify         = "DkimVerify"
	DkimMandatory      = "DkimMandatory" // Reject messages that aren't DKIM verified
	DkimPrivateKeyFile = "DkimPrivateKeyFile"
	DkimPublicKeyFile  = "DkimPublicKeyFile"
	DkimKeyBits        = "DkimKeyBits"

	// SPF
	//SpfVerify    = "SpfVerify"
	//SpfMandatory = "SpfMandatory" // Reject messages that aren't SPF verified

	// Web auth tokens
	JwtTokenSecretFile   = "JwtTokenSecretFile"
	JwtCookieName        = "JwtCookieName"
	CookieDomainOverride = "CookieDomainOverride"

	// Port our message sender will try to connect to MTAs on
	MtaSendPort = "MtaSendPort"
)

func SetupConfig() {
	viper.SetDefault(Domain, "example.com")
	viper.SetDefault(ServerName, "mail.example.com")

	viper.SetDefault(MsaAddress, ":587")
	viper.SetDefault(MtaAddress, ":25")
	viper.SetDefault(ImapAddress, ":143")
	viper.SetDefault(WebAdminAddress, ":443")
	viper.SetDefault(WebAdminUseTls, true)

	viper.SetDefault(MtaUseTls, true)
	viper.SetDefault(MsaUseTls, true)
	viper.SetDefault(ImapUseTls, true)
	viper.SetDefault(WebAdminUseTls, true)

	viper.SetDefault(CertificateMode, string(AutoCert))
	viper.SetDefault(AutoCertEmail, "admin@example.com")
	viper.SetDefault(AutoCertCacheDir, "keys")
	viper.SetDefault(CertificateFile, "/etc/letsencrypt/live/example.com/fullchain.pem")
	viper.SetDefault(KeyFile, "/etc/letsencrypt/live/example.com/privkey.pem")

	viper.SetDefault(DbDriverName, "sqlite3")
	viper.SetDefault(DbConnectionString, "henrymail.db")

	viper.SetDefault(MaxIdleSeconds, 300)
	viper.SetDefault(MaxMessageBytes, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipients, 50)
	viper.SetDefault(RetryCronSpec, "* * * * *") // every minute
	viper.SetDefault(RetryCount, 3)

	viper.SetDefault(AdminUsername, "admin")
	viper.SetDefault(AdminPassword, "") // Empty means it will be generated
	viper.SetDefault(DefaultMailboxes, []string{"INBOX", "Trash", "Sent", "Drafts"})

	viper.SetDefault(DkimSign, true)
	viper.SetDefault(DkimVerify, true)
	viper.SetDefault(DkimPrivateKeyFile, "keys/dkim-private.pem")
	viper.SetDefault(DkimPublicKeyFile, "keys/dkim-public.pem")
	viper.SetDefault(DkimKeyBits, 2048)

	viper.SetDefault(JwtTokenSecretFile, "keys/jwt-secret")
	viper.SetDefault(JwtCookieName, "henrymail_jwt_token")

	viper.SetDefault(DnsServer, "208.67.222.222:53") // OpenDNS

	// For running the full stack locally
	viper.SetDefault(FakeDns, false)
	viper.SetDefault(FakeDnsAddress, "127.0.0.1:2053")
	viper.SetDefault(MtaSendPort, ":25")
	viper.SetDefault(CookieDomainOverride, "")

	viper.SetConfigName("henrymail")
	viper.AddConfigPath("/etc/henrymail/")
	viper.AddConfigPath("$HOME/.henrymail")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Println(err)
	} else if err != nil {
		log.Fatal(err)
	}
}

func SetupResolver() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (conn net.Conn, e error) {
			d := net.Dialer{}
			return d.DialContext(ctx, network, GetString(DnsServer))
		},
	}
}
func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func GetCookieDomain() string {
	override := GetString(CookieDomainOverride)
	if override != "" {
		return override
	}
	return GetString(ServerName)
}
