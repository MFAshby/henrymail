package main

import (
	"database/sql"
	"henrymail/config"
	"henrymail/database"
	"henrymail/dkim"
	"henrymail/dns"
	"henrymail/imap"
	"henrymail/logic"
	"henrymail/process"
	"henrymail/smtp"
	"henrymail/web"
	"log"
	"math/rand"
)

//go:generate mkdir -p embedded
//go:generate embed

func main() {
	config.SetupConfig()
	config.SetupResolver()

	tlsConfig := config.GetTLSConfig()
	db := database.OpenDatabase()

	// submission agent processing chain
	var msaChain process.MsgProcessor = process.NewSender(db)
	if config.GetBool(config.DkimSign) {
		msaChain = process.NewDkimSigner(dkim.GetOrCreateDkim(), msaChain)
	}

	// transfer agent processing chain
	mtaChain := process.NewSaver(db)
	if config.GetBool(config.DkimVerify) {
		mtaChain = process.NewDkimVerifier(mtaChain)
	}

	// SPF checker
	// Virus scanner
	// Spam filter
	seedData(db)

	smtp.StartMsa(db, msaChain, tlsConfig)
	smtp.StartMta(db, mtaChain, tlsConfig)
	imap.StartImap(db, tlsConfig)
	web.StartWebAdmin(db, tlsConfig)

	if config.GetBool(config.FakeDns) {
		dns.StartFakeDNS(config.GetString(config.FakeDnsAddress), "udp")
	}

	// Wait for exit
	select {}
}

func seedData(db *sql.DB) {
	var pw string
	if config.GetString(config.AdminPassword) == "" {
		pw = randSeq(8)
	} else {
		pw = config.GetString(config.AdminPassword)
	}
	user, err := logic.NewUser(db, config.GetString(config.AdminUsername), pw, true)
	if err == nil {
		log.Printf("Generated admin user: %v password %v", user.Username, pw)
	}
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randSeq(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
