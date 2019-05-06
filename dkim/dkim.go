package dkim

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/gob"
	"henrymail/config"
	"henrymail/models"
	"log"
)

const (
	KeyName = "dkim"
)

func GetDkimRecordString(db *sql.DB) (string, error) {
	pk := GetOrCreateDkim(db)
	pkb, e := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	if e != nil {
		return "", e
	}
	buf := new(bytes.Buffer)
	_, e = base64.NewEncoder(base64.StdEncoding, buf).Write(pkb)
	if e != nil {
		return "", e
	}
	return "v=dkim1; k=rsa; p=" + buf.String(), nil
}

func GetOrCreateDkim(db *sql.DB) *rsa.PrivateKey {
	// Try the database
	gob.Register(rsa.PrivateKey{})
	var pk *rsa.PrivateKey
	dbKey, e := models.KeyByName(db, KeyName)
	if e == nil {
		e = gob.NewDecoder(bytes.NewReader(dbKey.Key)).Decode(&pk)
	}

	if e != nil {
		// Something is screwy, generate a new key
		log.Print(e)
		log.Println("Generating a new DKIM key")
		pk, e = rsa.GenerateKey(rand.Reader, config.GetInt(config.DkimKeyBits))
		if e != nil {
			// Can't generate a key, can't recover from this
			log.Fatal(e)
		}

		if dbKey == nil {
			// Assign a new db object if required
			dbKey = &models.Key{
				Name: KeyName,
			}
		}

		buffer := bytes.Buffer{}
		e = gob.NewEncoder(&buffer).Encode(pk)
		if e != nil {
			// Can't encode the key, can't recover from this
			log.Fatal(e)
		}
		dbKey.Key = buffer.Bytes()

		e = dbKey.Save(db)
		if e != nil {
			// Can't recover from this
			log.Fatal(e)
		}
	}
	return pk
}
