package config

import (
	"context"
	"github.com/spf13/viper"
	"log"
	"net"
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
	DnsServer = "DnsServer"

	// TLS
	MtaUseTls      = "MtaUseTls"
	MsaUseTls      = "MsaUseTls"
	ImapUseTls     = "ImapUseTls"
	WebAdminUseTls = "WebAdminUseTls"

	UseAutoCert = "UseAutoCert"
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
	DefaultMailboxes = "DefaultMailboxes"

	// DKIM
	DkimPrivateKeyFile = "DkimPrivateKeyFile"
	DkimPublicKeyFile  = "DkimPublicKeyFile"
	DkimKeyBits        = "DkimKeyBits"

	// Web auth tokens
	JwtTokenSecretFile = "JwtTokenSecretFile"
	JwtCookieName      = "JwtCookieName"
)

func SetupConfig() {
	viper.SetDefault(Domain, "example.com")

	viper.SetDefault(MsaAddress, ":1587")
	viper.SetDefault(MtaAddress, ":1025")
	viper.SetDefault(ImapAddress, ":1143")
	viper.SetDefault(WebAdminAddress, ":2003")
	viper.SetDefault(WebAdminUseTls, false)

	viper.SetDefault(UseAutoCert, true)
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
	viper.SetDefault(DefaultMailboxes, []string{"INBOX", "Trash", "Sent", "Drafts"})

	viper.SetDefault(DkimPrivateKeyFile, "keys/dkim-private.pem")
	viper.SetDefault(DkimPublicKeyFile, "keys/dkim-public.pem")
	viper.SetDefault(DkimKeyBits, 2048)

	viper.SetDefault(JwtTokenSecretFile, "keys/jwt-secret")
	viper.SetDefault(JwtCookieName, "henrymail_jwt_token")

	viper.SetDefault(DnsServer, "208.67.222.222:53") // OpenDNS

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
			return d.DialContext(ctx, "udp", GetString(DnsServer))
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
