package main

import "github.com/spf13/viper"

const (
	DomainKey     = "DOMAIN"
	SqlitePathKey = "DB_PATH"

	// Network
	MsaAddressKey      = "MSA_ADDRESS"
	MtaAddressKey      = "MTA_ADDRESS"
	ImapAddressKey     = "IMAP_ADDRESS"
	WebAdminAddressKey = "WEB_ADMIN_ADDRESS"

	// TLS
	UseAutoCertKey     = "USE_AUTO_CERT"
	AutoCertEmailKey   = "AUTO_CERT_EMAIL"
	AutoCertCacheDir   = "AUTO_CERT_CACHE_DIR"
	CertificateFileKey = "CERT_FILE"
	KeyFileKey         = "KEY_FILE"

	// Message sending stuff
	MaxIdleSecondsKey  = "MAX_IDLE_SECONDS"
	MaxMessageBytesKey = "MAX_MESSAGE_BYTES"
	MaxRecipientsKey   = "MAX_RECIPIENTS"
	RetryCronSpec      = "RETRY_CRON_SPEC"
	RetryCount         = "RETRY_COUNT"

	// Admin stuff
	AdminUsernameKey    = "ADMIN_USERNAME"
	AdminPasswordKey    = "ADMIN_PASSWORD"
	DefaultMailboxesKey = "DEFAULT_MAILBOXES"

	// DKIM
	DkimPrivateKeyFileKey = "DKIM_PRIVATE_KEY_FILE"
	DkimPublicKeyFileKey  = "DKIM_PUBLIC_KEY_FILE"
	DkimKeyBitsKey        = "DKIM_KEY_BITS"

	// Web auth tokens
	JwtTokenSecretFileKey = "JWT_TOKEN_SECRET_FILE"

	// DNS
	DnsServerKey = "DNS_SERVER"
)

func SetConfigDefaults() {
	viper.SetDefault(DomainKey, "henry-pi.site")
	viper.SetDefault(SqlitePathKey, "henrymail.db")
	viper.SetDefault(MsaAddressKey, ":1587")
	viper.SetDefault(MtaAddressKey, ":1025")
	viper.SetDefault(ImapAddressKey, ":1143")
	viper.SetDefault(WebAdminAddressKey, ":1443")

	viper.SetDefault(UseAutoCertKey, true)
	viper.SetDefault(AutoCertEmailKey, "martin@ashbysoft.com")
	viper.SetDefault(AutoCertCacheDir, "keys")
	viper.SetDefault(CertificateFileKey, "keys/henry-pi.site.crt")
	viper.SetDefault(KeyFileKey, "keys/henry-pi.site.key")

	viper.SetDefault(MaxIdleSecondsKey, 300)
	viper.SetDefault(MaxMessageBytesKey, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipientsKey, 50)
	viper.SetDefault(RetryCronSpec, "* * * * *") // every minute
	viper.SetDefault(RetryCount, 3)

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
