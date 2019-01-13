package main

import (
	"henrymail/config"
	"henrymail/database"
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

	pk := processors.GetOrCreateDkim()
	msaChain := processors.NewDkimSigner(pk, processors.NewSender(db))
	mtaChain := processors.NewDkimVerifier(processors.NewSaver(db))

	smtp.StartMsa(msaChain, login, tlsConfig)
	smtp.StartMta(mtaChain, tlsConfig)
	imap.StartImap(login, db, tlsConfig)
	web.StartWebAdmin(login, db, tlsConfig, &pk.PublicKey)

	// Setup admin user, domain keys if this if the first startup
	SeedData(login)

	// Wait for exit
	select {}
}

func SeedData(login database.Login) {
	//pw := randSeq(16)
	pw := "hello"
	usr, err := login.NewUser(config.GetString(config.AdminUsername)+"@"+config.GetString(config.Domain),
		pw, true)
	if err == nil {
		log.Printf("Generated admin user email: %v password %v", usr.Email, pw)
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
