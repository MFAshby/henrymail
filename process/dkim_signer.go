package process

import (
	"bytes"
	"crypto/rsa"
	"github.com/emersion/go-dkim"
	"henrymail/config"
)

type dkimSigner struct {
	dkim *rsa.PrivateKey
	next MsgProcessor
}

func (d dkimSigner) Process(msg *ReceivedMsg) error {
	options := &dkim.SignOptions{
		Domain:   config.GetString(config.Domain),
		Selector: "mx",
		Signer:   d.dkim,
	}
	var b bytes.Buffer
	e := dkim.Sign(&b, bytes.NewReader(msg.Content), options)
	if e != nil {
		return e
	}
	msg.Content = b.Bytes()
	return d.next.Process(msg)
}

func NewDkimSigner(dkim *rsa.PrivateKey, next MsgProcessor) MsgProcessor {
	return &dkimSigner{
		dkim: dkim,
		next: next,
	}
}
