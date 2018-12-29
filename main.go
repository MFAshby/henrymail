package main

func main() {
	SetConfigDefaults()

	tlsConfig := GetTLSConfig()
	database := NewDatabase()
	login := NewLogin(database)
	sender := NewSender(database)

	msaChain := sender
	mtaChain := NewDkimVerifier(NewSaver(database))

	StartMsa(msaChain, login, tlsConfig)
	StartMta(mtaChain, tlsConfig)
	StartImap(login, database, tlsConfig)
	StartWebAdmin(login, database, tlsConfig)

	// Test data setup
	ConfigureAdmin(login)

	// Wait for exit
	select {}
}

func ConfigureAdmin(login Login) {
	_, _ = login.NewUser(GetString(AdminUsernameKey)+"@"+GetString(DomainKey),
		GetString(AdminPasswordKey), true)
}
