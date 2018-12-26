package main

import (
	ev "github.com/asaskevich/EventBus"
)

func main() {
	SetConfigDefaults()

	database := NewDatabase()
	login := NewLogin(database)

	// Bus for posting async stuff
	bus := ev.New()

	// Log everything that is posted to the bus
	StartEventLogger(bus)

	// Various user facing services
	StartMsa(bus, login)
	StartMta(bus)
	StartImap(login, database)
	StartWebAdmin(bus, login, database)

	u1, _ := login.NewUser("martin@test.com", "12345")
	_, _ = login.NewUser("someone@test.com", "12345")
	m1, _ := database.InsertMailbox("INBOX", u1.Id)
	_, _ = database.InsertMailbox("Trash	", u1.Id)
	_, _ = database.InsertMessage([]byte("Hello world"), []string{"\\Recent"}, m1.Id)

	// Wait for exit
	select {}
}
