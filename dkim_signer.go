package main

import (
	"crypto"
)

type dkimSigner struct {
	pk   crypto.PrivateKey
	next MsgProcessor
}

func (d dkimSigner) Process(msg *ReceivedMsg) error {
	/*options := &dkim.SignOptions{
		Domain: GetString(DomainKey),
		Signer: d.pk,
	}
	dkim.Sign()*/
	return d.next.Process(msg)
}

func NewDkimSigner(pk crypto.PrivateKey, next MsgProcessor) MsgProcessor {
	return &dkimSigner{
		pk:   pk,
		next: next,
	}
}
