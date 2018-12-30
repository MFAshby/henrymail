package main

import "github.com/spf13/viper"

const (
	DomainKey     = "Domain"
	SqlitePathKey = "SqlitePath"

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

	// Message sending stuff
	MaxIdleSecondsKey  = "MaxIdleSeconds"
	MaxMessageBytesKey = "MaxMessageBytes"
	MaxRecipientsKey   = "MaxRecipients"
	RetryCronSpecKey   = "RetryCronSpec"
	RetryCountKey      = "RetryCount"

	// Admin stuff
	AdminUsernameKey    = "AdminUsername"
	AdminPasswordKey    = "AdminPassword"
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

func SetConfigDefaults() {
	viper.SetDefault(DomainKey, "mfashby.net")
	viper.SetDefault(SqlitePathKey, "henrymail.db")
	viper.SetDefault(MsaAddressKey, ":1587")
	viper.SetDefault(MtaAddressKey, ":1025")
	viper.SetDefault(ImapAddressKey, ":1143")
	viper.SetDefault(WebAdminAddressKey, ":2003")
	viper.SetDefault(WebAdminUseTlsKey, false)

	viper.SetDefault(UseAutoCertKey, false)
	viper.SetDefault(AutoCertEmailKey, "martin@ashbysoft.com")
	viper.SetDefault(AutoCertCacheDir, "keys")
	viper.SetDefault(CertificateFileKey, "/etc/letsencrypt/live/mail.mfashby.net/fullchaim.pem")
	viper.SetDefault(KeyFileKey, "/etc/letsencrypt/live/mail.mfashby.net/privkey.pem")

	viper.SetDefault(MaxIdleSecondsKey, 300)
	viper.SetDefault(MaxMessageBytesKey, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipientsKey, 50)
	viper.SetDefault(RetryCronSpecKey, "* * * * *") // every minute
	viper.SetDefault(RetryCountKey, 3)

	viper.SetDefault(AdminUsernameKey, "admin")
	viper.SetDefault(AdminPasswordKey, "iloveemail")
	viper.SetDefault(DefaultMailboxesKey, []string{"INBOX", "Trash", "Sent", "Drafts"})

	viper.SetDefault(DkimPrivateKeyFileKey, "keys/dkim-private.pem")
	viper.SetDefault(DkimPublicKeyFileKey, "keys/dkim-public.pem")
	viper.SetDefault(DkimKeyBitsKey, 2048)

	viper.SetDefault(JwtTokenSecretFileKey, "keys/jwt-secret")

	viper.SetDefault(DnsServerKey, "8.8.8.8:53")
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
