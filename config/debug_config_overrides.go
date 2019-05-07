// +build debug

package config

func SetupConfig() {
	SetString(AdminPassword, "admin")
	SetString(MsaAddress, ":1587")
	SetString(MtaAddress, ":1025")
	SetString(ImapAddress, ":1143")
	SetString(WebAdminAddress, ":2003")
	SetBool(MtaUseTls, false)
	SetBool(MsaUseTls, false)
	SetBool(ImapUseTls, false)
	SetBool(WebAdminUseTls, false)
	SetString(DnsServer, "127.0.0.1:2053")
	SetBool(FakeDns, true)
	SetString(MtaSendPort, ":1025")
	SetString(CookieDomainOverride, "localhost")
}
