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
	// StartMta()
	// StartImap()
	StartWebAdmin(bus, login)

	// Wait for exit
	select {}
}
