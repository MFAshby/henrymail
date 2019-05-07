package config

import (
	"context"
	"encoding/json"
	"henrymail/database"
	"henrymail/models"
	"log"
	"net"
	"strconv"
)

type CertMode string

const (
	AutoCert   CertMode = "AutoCert"
	Given               = "Given"
	SelfSigned          = "SelfSigned"
)

type ConfigKey string

type ConfigValues struct {
	DocString    string
	DefaultValue string
}

var configs = make(map[ConfigKey]ConfigValues)

func registerConfig(name, defaultValue, docstring string) ConfigKey {
	key := ConfigKey(name)
	configs[key] = ConfigValues{
		DocString:    docstring,
		DefaultValue: defaultValue,
	}
	return key
}

var (
	Domain = registerConfig("Domain", "example.com",
		`This is the domain for emails addresses that this server is responsible for
				  e.g. example.com`)

	ServerName = registerConfig("ServerName", "mail.example.com",
		`This is the DNS name of the email server. Note that it is different to the
				  domain, in case you want to host a website at the domain root.`)

	// iana.org/go/rfc6409
	MsaAddress = registerConfig("MsaAddress", ":587",
		`This is the address where the Message Submission Agent will listen.
				  The default value is the IANA recommended port, see http://www.`)

	MtaAddress = registerConfig("MtaAddress", ":25",
		`This is the address where the Message Transfer Agent will listen.
				  The default value is the IANA recommended port, see http://www.iana.org/go/rfc5321`)

	ImapAddress = registerConfig("ImapAddress", ":143",
		`This is the address where the IMAP server will listen
				 The default value is the IANA recommended port, see http://www.iana.org/go/rfc3501`)

	WebAdminAddress = registerConfig("WebAdminAddress", ":443",
		`This is the address where the web administration interface will listen.
				  The default value is the standard HTTPS port.`)

	DnsServer = registerConfig("DnsServer", "208.67.222.222:53",
		`Which DNS server should be used for sending emails and performing DNS health 
				 checks. Defaults to the OpenDNS Public DNS server: see https://use.opendns.com/`)

	FakeDns = registerConfig("FakeDns", strconv.FormatBool(false),
		`It is desirable during development and testing to spoof DNS records, without
				  having to expose the development environment to the internet. These settings
				  enable an in-built DNS server to provide this facility. They should not
				  be used in production environments.`)

	FakeDnsAddress = registerConfig("FakeDnsAddress", "127.0.0.1:2053",
		`The address that the fake DNS server binds to. See FakeDns`)

	MtaUseTls = registerConfig("MtaUseTls", strconv.FormatBool(true),
		`Defines whether to use Transport Layer Security for the message transfer agent
				 It is recommended that you leave this enabled.
				 It should only be disabled for convenience during application development.`)

	MsaUseTls = registerConfig("MsaUseTls", strconv.FormatBool(true),
		`Defines whether to use Transport Layer Security for the message submission agent
				 It is recommended that you leave this enabled.
				 It should only be disabled for convenience during application development.`)
	ImapUseTls = registerConfig("ImapUseTls", strconv.FormatBool(true),
		`Defines whether to use Transport Layer Security for the imap server
				 It is recommended that you leave this enabled.
				 It should only be disabled for convenience during application development.`)

	WebAdminUseTls = registerConfig("WebAdminUseTls", strconv.FormatBool(true),
		`Defines whether to use Transport Layer Security for the web administration
				 interface. It is recommended to leave this enabled, unless you are running
				 henrymail behind a reverse proxy like nginx, which terminates the TLS
				 connection for you.`)

	CertificateMode = registerConfig("CertificateMode", string(AutoCert),
		`Henrymail supports 3 modes for certificates: AutoCert, Given, and SelfSigned
				  AutoCert will automatically obtain certificates from LetsEncrypt,
				    see https://letsencrypt.org/. This is the most convenient setting if you
			        are running a standalone henrymail server.
				  Given will use an existing certificate on disk. This is advisable if you
					already have a certificate, and you share it with other applications (e.g.
					a web server)
				  SelfSigned will generate a self-signed certificate (not advisable in production
					systems)`)

	AutoCertEmail = registerConfig("AutoCertEmail", "admin@example.com",
		`If CertificateMode is set to AutoCert, then this email address is used by
				  LetsEncrypt to send you warnings if your certificate will expire, or if there
				  is some other problem.`)

	//TODO remove this, it should also be in the database.
	AutoCertCacheDir = registerConfig("AutoCertCacheDir", "keys",
		`This is the directory in which henrymail will store automatically generated
				  certificates.`)

	CertificateFile = registerConfig("CertificateFile", "/etc/letsencrypt/live/example.com/fullchain.pem",
		`If CertificateMode is set to Given, then this should be the path to the
				 public certificate. The certificate must be issued for the server
				 set in the ServerName setting.`)

	KeyFile = registerConfig("KeyFile", "/etc/letsencrypt/live/example.com/privkey.pem",
		`If CertificateMode is set to Given, then this should be the path to the
				  private key.`)

	MaxIdleSeconds = registerConfig("MaxIdleSeconds", strconv.Itoa(300),
		`Controls how long SMTP clients are allowed to be idle before the connection
				 is terminated.`)

	MaxMessageBytes = registerConfig("MaxMessageBytes", strconv.Itoa(1024^2),
		`Controls the maximum message size that the SMTP server(s) will accept`)

	MaxRecipients = registerConfig("MaxRecipients", strconv.Itoa(50),
		`Controls the maximum number of recipients the the SMTP server(s) will accept`)

	RetryCronSpec = registerConfig("RetryCronSpec", "* * * * *",
		`Controls how frequently retries are attempted, if an email fails to send due
				  to server down time. The format is described here:
				  https://godoc.org/github.com/robfig/cron`)

	RetryCount = registerConfig("RetryCount", strconv.Itoa(3),
		`How many retries are attempted before the server stops trying and sends a
				 failure notification instead.`)

	//TODO remove this
	AdminUsername = registerConfig("AdminUsername", "admin",
		`When the server is first started, an administrator user is generated.
				 This setting controls the username`)

	//TODO remove this
	AdminPassword = registerConfig("AdminPassword", "remove this", "")

	DefaultMailboxes = registerConfig("DefaultMailboxes", `["INBOX", "Trash", "Sent", "Drafts"]`,
		`The mailboxes which will be configured for each new account, by default.`)

	DkimSign = registerConfig("DkimSign", strconv.FormatBool(true),
		`This setting controls whether emails sent from henrymail are signed with DKIM.`)

	DkimVerify = registerConfig("DkimVerify", strconv.FormatBool(true),
		`This setting controls is incoming emails DKIM signatures should be verified.
				  Dkim records that are present but incorrect will cause the email to be rejected.`)

	DkimMandatory = registerConfig("DkimMandatory", strconv.FormatBool(false),
		`This setting controls if incoming emails MUST be signed with DKIM.
				  Enabling this setting can reduce spam, however it will also reject any
				   emails from poorly configured email domains.`)

	DkimKeyBits = registerConfig("DkimKeyBits", strconv.Itoa(2048),
		`The length of the DKIM signing key. 2048 is a safe default. 1024 is the minimum
				  recommended setting.`)

	SpfVerify = registerConfig("SpfVerify", strconv.FormatBool(true),
		`This setting controls whether SPF records should be verified for incoming emails
				  SPF records that are present but incorrect will cause the email to be rejected.`)

	SpfMandatory = registerConfig("SpfMandatory", strconv.FormatBool(false),
		`This setting controls whether SPF records are mandatory for incoming emails.
				 Enabling this setting can reduce spam, however it will also reject any
				 emails from poorly configured email domains.`)

	JwtCookieName = registerConfig("JwtCookieName", "henrymail_jwt_token",
		`This setting controls the name of the cookie that's stored in users' browsers for authentication`)

	CookieDomainOverride = registerConfig("CookieDomainOverride", "",
		`This setting controls what domain is used for the authentication cookie. This should
				  only be used during development, when the server name setting may differ from the
				  actual server name. Use in a production environment will cause the web admin interface
				  to stop working.`)

	MtaSendPort = registerConfig("MtaSendPort", ":25",
		`This setting controls which port our server will connect to in order to pass on
				  submitted emails. This should only be changed for development purposes when you
				  wish to test emails looping back to your own server locally. Changing this in
				  production will cause all outgoing email to fail.`)
)

func SetupResolver() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (conn net.Conn, e error) {
			d := net.Dialer{}
			return d.DialContext(ctx, network, GetString(DnsServer))
		},
	}
}

func GetString(key ConfigKey) string {
	cfg, e := models.ConfigByName(database.DB, string(key))
	if e == nil {
		return cfg.Value
	} else {
		return configs[key].DefaultValue
	}
}

func SetString(key ConfigKey, value string) {
	stringKey := string(key)
	config, e := models.ConfigByName(database.DB, stringKey)
	if e != nil {
		config = &models.Config{
			Name: stringKey,
		}
	}
	config.Value = value
	e = config.Save(database.DB)
	if e != nil {
		// Failing to set a config value is... bad :(
		log.Fatal(e)
	}
}

func GetInt(key ConfigKey) int {
	i, e := strconv.Atoi(GetString(key))
	if e != nil {
		log.Fatal(e)
	}
	return i
}

func GetBool(key ConfigKey) bool {
	b, e := strconv.ParseBool(GetString(key))
	if e != nil {
		log.Fatal(e)
	}
	return b
}

func SetBool(key ConfigKey, value bool) {
	SetString(key, strconv.FormatBool(value))
}

func GetStringSlice(key ConfigKey) []string {
	var res []string
	e := json.Unmarshal([]byte(GetString(key)), &res)
	if e != nil {
		log.Fatal(e)
	}
	return res
}

func GetCookieDomain() string {
	override := GetString(CookieDomainOverride)
	if override != "" {
		return override
	}
	return GetString(ServerName)
}
