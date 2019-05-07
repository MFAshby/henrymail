package main

import (
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
	database.OpenDatabase()
	config.SetupConfig()
	config.SetupResolver()

	tlsConfig := config.GetTLSConfig()

	// submission agent processing chain
	var msaChain process.MsgProcessor = process.NewSender()
	if config.GetBool(config.DkimSign) {
		msaChain = process.NewDkimSigner(dkim.GetOrCreateDkim(), msaChain)
	}

	// transfer agent processing chain
	mtaChain := process.NewSaver()
	if config.GetBool(config.DkimVerify) {
		mtaChain = process.NewDkimVerifier(mtaChain)
	}

	// SPF checker
	// Virus scanner
	// Spam filter
	seedData()

	smtp.StartMsa(msaChain, tlsConfig)
	smtp.StartMta(mtaChain, tlsConfig)
	imap.StartImap(tlsConfig)
	web.StartWebAdmin(tlsConfig)

	if config.GetBool(config.FakeDns) {
		dns.StartFakeDNS(config.GetString(config.FakeDnsAddress), "udp")
	}

	// Wait for exit
	select {}
}

func seedData() {
	var pw string
	if config.GetString(config.AdminPassword) == "" {
		pw = randSeq(8)
	} else {
		pw = config.GetString(config.AdminPassword)
	}
	user, err := logic.NewUser(database.DB, config.GetString(config.AdminUsername), pw, true)
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
