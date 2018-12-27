package main

import (
	"crypto/tls"
	"log"
)

func main() {
	SetConfigDefaults()

	certificate, e := tls.LoadX509KeyPair("henry-pi.site.crt", "henry-pi.site.key")
	if e != nil {
		log.Fatal(e)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}

	database := NewDatabase()
	login := NewLogin(database)
	sender := NewSender(database)

	msaChain := NewLogger(sender)
	mtaChain := NewLogger(NewSaver(database))

	StartMsa(msaChain, login, config)
	StartMta(mtaChain, config)
	StartImap(login, database, config)
	StartWebAdmin(login, database)

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
