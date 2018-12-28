package main

func main() {
	SetConfigDefaults()

	tlsConfig := GetTLSConfig()
	database := NewDatabase()
	login := NewLogin(database)
	sender := NewSender(database)

	msaChain := sender
	mtaChain := NewSaver(database)

	StartMsa(msaChain, login, tlsConfig)
	StartMta(mtaChain, tlsConfig)
	StartImap(login, database, tlsConfig)
	StartWebAdmin(login, database, tlsConfig)

	// Test data setup
	u1, e := login.NewUser("martin@henry-pi.site", "12345")
	if e == nil {
		_, _ = database.InsertMailbox("INBOX", u1.Id)
		_, _ = database.InsertMailbox("Trash", u1.Id)
		_, _ = database.InsertMailbox("Sent", u1.Id)
		_, _ = database.InsertMailbox("Drafts", u1.Id)
	}

	// Wait for exit
	select {}
}
