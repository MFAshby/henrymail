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
 * Accepts new mail from our own users for sending
 */
func StartMsa(proc MsgProcessor, lg Login) {
	be := &sbe{
		proc: proc,
		lg:   lg,
	}
	s := smtp.NewServer(be)
	s.Addr = GetString(MsaAddressKey)
	s.Domain = GetString(DomainKey)
	s.MaxIdleSeconds = GetInt(MaxIdleSecondsKey)
	s.MaxMessageBytes = GetInt(MaxMessageBytesKey)
	s.MaxRecipients = GetInt(MaxRecipientsKey)
	s.AllowInsecureAuth = GetBool(AllowInsecureAuthKey)
	s.Debug = os.Stdout
	go func() {
		log.Println("Starting mail submission agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type sbe struct {
	proc MsgProcessor
	lg   Login
}

func (b *sbe) Login(username, password string) (smtp.User, error) {
	user, e := b.lg.Login(username, password)
	if e != nil {
		return nil, e
	}
	return &sus{
		proc: b.proc,
		u:    user,
	}, nil
}

func (b *sbe) AnonymousLogin() (smtp.User, error) {
	return nil, smtp.ErrAuthRequired
}

type sus struct {
	proc MsgProcessor
	u    *Usr
}

func (u *sus) Send(from string, to []string, r io.Reader) error {
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

func (*sus) Logout() error {
	return nil
}
