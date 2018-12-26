package main

import (
	"log"
)

func main() {
	SetConfigDefaults()

	database := NewDatabase()
	login := NewLogin(database)

	// Define processing chains
	sender := NewSender(database)
	StartMsa(NewLogger(sender), login)
	StartMta(NewLogger(NewSaver(database)))
	StartImap(login, database)
	StartWebAdmin(login, database)

	err := sender.StartRetries()
	if err != nil {
		log.Fatal(err)
	}

	// Test data setup
	u1, _ := login.NewUser("martin@test.com", "12345")
	_, _ = login.NewUser("someone@test.com", "12345")
	m1, _ := database.InsertMailbox("INBOX", u1.Id)
	_, _ = database.InsertMailbox("Trash	", u1.Id)
	_, _ = database.InsertMessage([]byte("Hello world"), []string{"\\Recent"}, m1.Id)

	// Wait for exit
	select {}
}
