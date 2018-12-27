package main

import (
	"bytes"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"io"
	"io/ioutil"
	"log"
	"os"
)

/**
 * Accepts new mail from other servers
 */
func StartMta(proc MsgProcessor) {
	b := &tbe{
		proc: proc,
	}
	s := smtp.NewServer(b)
	s.Addr = GetString(MtaAddressKey)
	s.Domain = GetString(DomainKey)
	s.MaxIdleSeconds = GetInt(MaxIdleSecondsKey)
	s.MaxMessageBytes = GetInt(MaxMessageBytesKey)
	s.MaxRecipients = GetInt(MaxRecipientsKey)
	s.AllowInsecureAuth = GetBool(AllowInsecureAuthKey)
	s.Debug = os.Stdout
	go func() {
		log.Println("Starting mail transfer agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type tbe struct {
	proc MsgProcessor
	lg   Login
}

func (b *tbe) Login(username, password string) (smtp.User, error) {
	return nil, smtp.ErrAuthUnsupported
}

func (b *tbe) AnonymousLogin() (smtp.User, error) {
	return &tus{proc: b.proc}, nil
}

type tus struct {
	proc MsgProcessor
}

// TODO fix code duplication here
func (u *tus) Send(from string, to []string, r io.Reader) error {
	content, e := ioutil.ReadAll(r)
	// Check we can read all the content
	if e != nil {
		return e
	}

	// Check we can parse it as a spec compliant message
	if _, e := message.Read(bytes.NewBuffer(content)); e != nil {
		return e
	}

	// Pass it on
	return u.proc.Process(&ReceivedMsg{
		From:    from,
		To:      to,
		Content: content,
	})
}

func (*tus) Logout() error {
	return nil
}
