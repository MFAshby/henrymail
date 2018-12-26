package main

import (
	ev "github.com/asaskevich/EventBus"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"io"
	"log"
)

/**
 * Accepts new mail from other servers
 */
func StartMta(bus ev.Bus) {
	b := &tbe{
		bus: bus,
	}
	s := smtp.NewServer(b)
	s.Addr = GetString(MtaAddressKey)
	s.Domain = GetString(DomainKey)
	s.MaxIdleSeconds = GetInt(MaxIdleSecondsKey)
	s.MaxMessageBytes = GetInt(MaxMessageBytesKey)
	s.MaxRecipients = GetInt(MaxRecipientsKey)
	s.AllowInsecureAuth = GetBool(AllowInsecureAuthKey)
	go func() {
		log.Println("Starting mail transfer agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type tbe struct {
	bus ev.Bus
	lg  Login
}

func (b *tbe) Login(username, password string) (smtp.User, error) {
	return nil, smtp.ErrAuthUnsupported
}

func (b *tbe) AnonymousLogin() (smtp.User, error) {
	return &tus{bus: b.bus}, nil
}

type tus struct {
	bus ev.Bus
}

func (u *tus) Send(from string, to []string, r io.Reader) error {
	if msg, e := message.Read(r); e != nil {
		return e
	} else {
		u.bus.Publish(MailSubmitted, msg)
		return nil
	}
}

func (*tus) Logout() error {
	return nil
}
