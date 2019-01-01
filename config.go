package main

import (
	"context"
	"github.com/spf13/viper"
	"log"
	"net"
)

const (
	DomainKey = "Domain"

	// Network
	MsaAddressKey      = "MsaAddress"
	MtaAddressKey      = "MtaAddres"
	ImapAddressKey     = "ImapAddress"
	WebAdminAddressKey = "WebAdminAddress"
	WebAdminUseTlsKey  = "WebAdminUseTls"

	// TLS
	UseAutoCertKey     = "UseAutoCert"
	AutoCertEmailKey   = "AutCertEmail"
	AutoCertCacheDir   = "AuthCertCacheDir"
	CertificateFileKey = "CertificateFile"
	KeyFileKey         = "KeyFile"

	// Database
	DbDriverNameKey       = "DbDriverName"
	DbConnectionStringKey = "DbConnectionString"

	// Message sending stuff
	MaxIdleSecondsKey  = "MaxIdleSeconds"
	MaxMessageBytesKey = "MaxMessageBytes"
	MaxRecipientsKey   = "MaxRecipients"
	RetryCronSpecKey   = "RetryCronSpec"
	RetryCountKey      = "RetryCount"

	// Admin stuff
	AdminUsernameKey    = "AdminUsername"
	DefaultMailboxesKey = "DefaultMailboxes"

	// DKIM
	DkimPrivateKeyFileKey = "DkimPrivateKeyFile"
	DkimPublicKeyFileKey  = "DkimPublicKeyFile"
	DkimKeyBitsKey        = "DkimKeyBits"

	// Web auth tokens
	JwtTokenSecretFileKey = "JwtTokenSecretFile"

	// DNS
	DnsServerKey = "DnsServer"
)

func SetupConfig() {
	viper.SetDefault(DomainKey, "example.com")

	viper.SetDefault(MsaAddressKey, ":1587")
	viper.SetDefault(MtaAddressKey, ":1025")
	viper.SetDefault(ImapAddressKey, ":1143")
	viper.SetDefault(WebAdminAddressKey, ":2003")
	viper.SetDefault(WebAdminUseTlsKey, false)

	viper.SetDefault(UseAutoCertKey, true)
	viper.SetDefault(AutoCertEmailKey, "admin@example.com")
	viper.SetDefault(AutoCertCacheDir, "keys")
	viper.SetDefault(CertificateFileKey, "/etc/letsencrypt/live/example.com/fullchain.pem")
	viper.SetDefault(KeyFileKey, "/etc/letsencrypt/live/example.com/privkey.pem")

	viper.SetDefault(DbDriverNameKey, "sqlite3")
	viper.SetDefault(DbConnectionStringKey, "henrymail.db")

	viper.SetDefault(MaxIdleSecondsKey, 300)
	viper.SetDefault(MaxMessageBytesKey, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipientsKey, 50)
	viper.SetDefault(RetryCronSpecKey, "* * * * *") // every minute
	viper.SetDefault(RetryCountKey, 3)

	viper.SetDefault(AdminUsernameKey, "admin")
	viper.SetDefault(DefaultMailboxesKey, []string{"INBOX", "Trash", "Sent", "Drafts"})

	viper.SetDefault(DkimPrivateKeyFileKey, "keys/dkim-private.pem")
	viper.SetDefault(DkimPublicKeyFileKey, "keys/dkim-public.pem")
	viper.SetDefault(DkimKeyBitsKey, 2048)

	viper.SetDefault(JwtTokenSecretFileKey, "keys/jwt-secret")

	viper.SetDefault(DnsServerKey, "8.8.8.8:53")

	viper.SetConfigName("henrymail")
	viper.AddConfigPath("/etc/henrymail/")
	viper.AddConfigPath("$HOME/.henrymail")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
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
			return d.DialContext(ctx, "udp", GetString(DnsServerKey))
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
