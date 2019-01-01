package main

import (
	"log"
	"math/rand"
)

func main() {
	SetupConfig()
	SetupResolver()

	tlsConfig := GetTLSConfig()
	database := NewDatabase()
	login := NewLogin(database)

	pk := GetOrCreateDkim()
	msaChain := NewDkimSigner(pk, NewSender(database))
	mtaChain := NewDkimVerifier(NewSaver(database))

	StartMsa(msaChain, login, tlsConfig)
	StartMta(mtaChain, tlsConfig)
	StartImap(login, database, tlsConfig)
	StartWebAdmin(login, database, tlsConfig, &pk.PublicKey)

	// Setup admin user, domain keys if this if the first startup
	SeedData(login)

	// Wait for exit
	select {}
}

func SeedData(login Login) {
	pw := randSeq(16)
	usr, err := login.NewUser(GetString(AdminUsernameKey)+"@"+GetString(DomainKey),
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
