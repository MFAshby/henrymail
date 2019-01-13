package processors

import (
	"bytes"
	"github.com/emersion/go-dkim"
	"henrymail/model"
)

type dkimVerifier struct {
	next MsgProcessor
}

func (d dkimVerifier) Process(msg *model.ReceivedMsg) error {
	v, e := dkim.Verify(bytes.NewReader(msg.Content))
	if e != nil {
		return e
	}
	msg.Verifications = v
	return d.next.Process(msg)
}

func NewDkimVerifier(next MsgProcessor) MsgProcessor {
	return &dkimVerifier{
		next: next,
	}
}
