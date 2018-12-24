package main

import (
	ev "github.com/asaskevich/EventBus"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"io"
	"log"
)

func StartMsa(bus ev.Bus, lg Login) {
	be := &be{
		bus: bus,
		lg:  lg,
	}
	s := smtp.NewServer(be)
	s.Addr = GetString(MsaAddressKey)
	s.Domain = GetString(DomainKey)
	s.MaxIdleSeconds = GetInt(MaxIdleSecondsKey)
	s.MaxMessageBytes = GetInt(MaxMessageBytesKey)
	s.MaxRecipients = GetInt(MaxRecipientsKey)
	s.AllowInsecureAuth = GetBool(AllowInsecureAuthKey)
	go func() {
		log.Println("Starting mail submission agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type be struct {
	bus ev.Bus
	lg  Login
}

func (bkd *be) Login(username, password string) (smtp.User, error) {
	user, e := bkd.lg.Login(username, password)
	if e != nil {
		return nil, e
	}
	return &us{
		bus: bkd.bus,
		u:   user,
	}, nil
}

func (bkd *be) AnonymousLogin() (smtp.User, error) {
	return nil, smtp.ErrAuthRequired
}

type us struct {
	bus ev.Bus
	u   *User
}

func (u us) Send(from string, to []string, r io.Reader) error {
	if msg, e := message.Read(r); e != nil {
		return e
	} else {
		u.bus.Publish(MailSubmitted, msg)
		return nil
	}
}

func (us) Logout() error {
	return nil
}
