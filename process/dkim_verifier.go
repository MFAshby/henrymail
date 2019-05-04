package process

import (
	"bytes"
	"github.com/emersion/go-dkim"
	"github.com/pkg/errors"
	"henrymail/config"
)

type dkimVerifier struct {
	next MsgProcessor
}

func (d dkimVerifier) Process(msg *ReceivedMsg) error {
	v, e := dkim.Verify(bytes.NewReader(msg.Content))
	if e != nil {
		return e
	}
	msg.Verifications = v

	if config.GetBool(config.DkimMandatory) && len(msg.Verifications) == 0 {
		return errors.New("dkim verification failed")
	}

	return d.next.Process(msg)
}

func NewDkimVerifier(next MsgProcessor) MsgProcessor {
	return &dkimVerifier{
		next: next,
	}
}
