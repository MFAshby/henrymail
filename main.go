package main

func main() {
	SetConfigDefaults()

	tlsConfig := GetTLSConfig()
	database := NewDatabase()
	login := NewLogin(database)
	sender := NewSender(database)

	pk := GetOrCreateDkim()
	msaChain := NewDkimSigner(pk, sender)
	mtaChain := NewDkimVerifier(NewSaver(database))

	StartMsa(msaChain, login, tlsConfig)
	StartMta(mtaChain, tlsConfig)
	StartImap(login, database, tlsConfig)
	StartWebAdmin(login, database, tlsConfig)

	// Setup admin user, domain keys if this if the first startup
	SeedData(login)

	// Wait for exit
	select {}
}

func SeedData(login Login) {
	_, _ = login.NewUser(GetString(AdminUsernameKey)+"@"+GetString(DomainKey),
		GetString(AdminPasswordKey), true)
}
