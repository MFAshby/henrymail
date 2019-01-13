package processors

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/emersion/go-dkim"
	"henrymail/config"
	"henrymail/model"
	"io/ioutil"
	"log"
	"os"
	"path"
)

type dkimSigner struct {
	pk   *rsa.PrivateKey
	next MsgProcessor
}

func (d dkimSigner) Process(msg *model.ReceivedMsg) error {
	options := &dkim.SignOptions{
		Domain:   config.GetString(config.DomainKey),
		Selector: "mx",
		Signer:   d.pk,
	}
	var b bytes.Buffer
	e := dkim.Sign(&b, bytes.NewReader(msg.Content), options)
	if e != nil {
		return e
	}
	msg.Content = b.Bytes()
	return d.next.Process(msg)
}

func NewDkimSigner(pk *rsa.PrivateKey, next MsgProcessor) MsgProcessor {
	return &dkimSigner{
		pk:   pk,
		next: next,
	}
}

func GetOrCreateDkim() *rsa.PrivateKey {
	var privKey *rsa.PrivateKey
	var e error

	privKeyFileName := config.GetString(config.DkimPrivateKeyFileKey)
	pubKeyFileName := config.GetString(config.DkimPublicKeyFileKey)
	privKeyBytes, e1 := ioutil.ReadFile(privKeyFileName)
	pubKeyBytes, e2 := ioutil.ReadFile(pubKeyFileName)

	if os.IsNotExist(e1) && os.IsNotExist(e2) {
		_ = os.MkdirAll(path.Dir(privKeyFileName), 0700)
		_ = os.MkdirAll(path.Dir(pubKeyFileName), 0700)

		privKey, e = rsa.GenerateKey(rand.Reader, config.GetInt(config.DkimKeyBitsKey))
		if e != nil {
			log.Fatal(e)
		}
		e = ioutil.WriteFile(privKeyFileName,
			ExportRsaPrivateKeyAsPem(privKey),
			0700)
		if e != nil {
			log.Fatal(e)
		}

		pubKey, e := ExportRsaPublicKeyAsPem(&privKey.PublicKey)
		if e != nil {
			log.Fatal(e)
		}

		e = ioutil.WriteFile(pubKeyFileName, pubKey, 0700)
		if e != nil {
			log.Fatal(e)
		}
	} else if e1 != nil {
		log.Fatal(e1)
	} else if e2 != nil {
		log.Fatal(e2)
	} else {
		privKey, e = ParseRsaPrivateKeyFromPem(privKeyBytes)
		if e != nil {
			log.Fatal(e)
		}
		publicKey, e := ParseRsaPublicKeyFromPem(pubKeyBytes)
		if e != nil {
			log.Fatal(e)
		}
		privKey.PublicKey = *publicKey
	}
	return privKey
}

func ExportRsaPrivateKeyAsPem(pk *rsa.PrivateKey) []byte {
	pkb := x509.MarshalPKCS1PrivateKey(pk)
	pkp := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: pkb,
		},
	)
	return pkp
}

func ParseRsaPrivateKeyFromPem(pkp []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pkp)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func ExportRsaPublicKeyAsPem(pk *rsa.PublicKey) ([]byte, error) {
	pkb, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return nil, err
	}
	pkp := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pkb,
		},
	)
	return pkp, nil
}

func ParseRsaPublicKeyFromPem(pkp []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pkp)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		return nil, errors.New("key type is not RSA")
	}
}
