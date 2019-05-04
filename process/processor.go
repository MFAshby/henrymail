package process

import (
	"github.com/emersion/go-dkim"
	"time"
)

type ReceivedMsg struct {
	From      string
	To        []string
	Content   []byte
	Timestamp time.Time

	Verifications []*dkim.Verification
}

type MsgProcessor interface {
	Process(*ReceivedMsg) error
}
