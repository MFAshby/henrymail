package main

import (
	"henrymail/config"
	"henrymail/database"
	"henrymail/dkim"
	"henrymail/imap"
	"henrymail/processors"
	"henrymail/smtp"
	"henrymail/web"
	"log"
	"math/rand"
)

func main() {
	config.SetupConfig()
	config.SetupResolver()

	tlsConfig := config.GetTLSConfig()
	db := database.NewDatabase()
	login := database.NewLogin(db)

	// submission agent processing chain
	var msaChain processors.MsgProcessor = processors.NewSender(db)
	if config.GetBool(config.DkimSign) {
		msaChain = processors.NewDkimSigner(dkim.GetOrCreateDkim(), msaChain)
	}

	// transfer agent processing chain
	mtaChain := processors.NewSaver(db)
	if config.GetBool(config.DkimVerify) {
		mtaChain = processors.NewDkimVerifier(mtaChain)
	}

	// TODO
	// SPF checker
	// Virus scanner
	// Spam filter

	SeedData(login)

	smtp.StartMsa(msaChain, login, tlsConfig)
	smtp.StartMta(mtaChain, tlsConfig)
	imap.StartImap(login, db, tlsConfig)
	web.StartWebAdmin(login, db, tlsConfig)

	if config.GetBool(config.FakeDns) {
		StartFakeDns(config.GetString(config.FakeDnsAddress), "udp")
	}

	// Wait for exit
	select {}
}

func SeedData(login database.Login) {
	var pw string
	if config.GetString(config.AdminPassword) == "" {
		pw = randSeq(16)
	} else {
		pw = config.GetString(config.AdminPassword)
	}

	usr, err := login.NewUser(config.GetString(config.AdminUsername),
		pw, true)
	if err == nil {
		log.Printf("Generated admin user: %v password %v", usr.Username, pw)
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
