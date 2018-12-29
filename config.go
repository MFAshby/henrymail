package main

import "github.com/spf13/viper"

const (
	DomainKey          = "DOMAIN"
	SqlitePathKey      = "DB_PATH"
	MsaAddressKey      = "MSA_ADDRESS"
	MtaAddressKey      = "MTA_ADDRESS"
	ImapAddressKey     = "IMAP_ADDRESS"
	WebAdminAddressKey = "WEB_ADMIN_ADDRESS"

	// TLS stuff
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
	viper.SetDefault(AutoCertCacheDir, "certs")
	viper.SetDefault(CertificateFileKey, "certs/henry-pi.site.crt")
	viper.SetDefault(KeyFileKey, "certs/henry-pi.site.key")

	viper.SetDefault(MaxIdleSecondsKey, 300)
	viper.SetDefault(MaxMessageBytesKey, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipientsKey, 50)
	viper.SetDefault(RetryCronSpec, "* * * * *") // every minute
	viper.SetDefault(RetryCount, 3)

	viper.SetDefault(AdminUsernameKey, "admin")
	viper.SetDefault(AdminPasswordKey, "iloveemail")
	viper.SetDefault(DefaultMailboxesKey, []string{"INBOX", "Trash", "Sent", "Drafts"})
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
