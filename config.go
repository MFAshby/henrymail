package main

import "github.com/spf13/viper"

const (
	SqlitePathKey        = "DB_PATH"
	MsaAddressKey        = "MSA_ADDRESS"
	MtaAddressKey        = "MTA_ADDRESS"
	ImapAddressKey       = "IMAP_ADDRESS"
	DomainKey            = "DOMAIN"
	MaxIdleSecondsKey    = "MAX_IDLE_SECONDS"
	MaxMessageBytesKey   = "MAX_MESSAGE_BYTES"
	MaxRecipientsKey     = "MAX_RECIPIENTS"
	AllowInsecureAuthKey = "ALLOW_INSECURE_AUTH"
	WebAdminAddressKey   = "WEB_ADMIN_ADDRESS"
	RetryCronSpec        = "RETRY_CRON_SPEC"
)

func SetConfigDefaults() {
	viper.SetDefault(SqlitePathKey, "henrymail.db")
	viper.SetDefault(MsaAddressKey, ":1587")
	viper.SetDefault(MtaAddressKey, ":1025")
	viper.SetDefault(ImapAddressKey, ":1143")
	viper.SetDefault(WebAdminAddressKey, ":1080")
	viper.SetDefault(DomainKey, "mfashby.net")
	viper.SetDefault(MaxIdleSecondsKey, 300)
	viper.SetDefault(MaxMessageBytesKey, 1024*1024) // 1MB
	viper.SetDefault(MaxRecipientsKey, 50)
	viper.SetDefault(AllowInsecureAuthKey, true)
	viper.SetDefault(RetryCronSpec, "* * * * *") // every minute
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
