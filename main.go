package main

func main() {
	SetConfigDefaults()

	database := NewDatabase()
	login := NewLogin(database)
	sender := NewSender(database)

	msaChain := NewLogger(sender)
	mtaChain := NewLogger(NewSaver(database))

	StartMsa(msaChain, login)
	StartMta(mtaChain)
	StartImap(login, database)
	StartWebAdmin(login, database)
	sender.StartRetries()

	// Test data setup
	u1, _ := login.NewUser("martin@test.com", "12345")
	_, _ = login.NewUser("someone@test.com", "12345")
	m1, _ := database.InsertMailbox("INBOX", u1.Id)
	_, _ = database.InsertMailbox("Trash", u1.Id)
	_, _ = database.InsertMailbox("Sent", u1.Id)
	_, _ = database.InsertMessage([]byte("Hello world"), []string{"\\Recent"}, m1.Id)

	// Wait for exit
	select {}
}
