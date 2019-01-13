package smtp

import (
	"bytes"
	"crypto/tls"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"henrymail/config"
	"henrymail/database"
	"henrymail/model"
	"henrymail/processors"
	"io"
	"io/ioutil"
	"log"
	"os"
)

/**
 * Accepts new mail from our own users for sending
 */
func StartMsa(proc processors.MsgProcessor, lg database.Login, tls *tls.Config) {
	be := &sbe{
		proc: proc,
		lg:   lg,
	}
	s := smtp.NewServer(be)
	s.Addr = config.GetString(config.MsaAddress)
	s.Domain = config.GetString(config.ServerName)
	s.MaxIdleSeconds = config.GetInt(config.MaxIdleSeconds)
	s.MaxMessageBytes = config.GetInt(config.MaxMessageBytes)
	s.MaxRecipients = config.GetInt(config.MaxRecipients)
	s.AllowInsecureAuth = !config.GetBool(config.MsaUseTls)
	s.Debug = os.Stdout
	s.TLSConfig = tls
	go func() {
		log.Println("Starting mail submission agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type sbe struct {
	proc processors.MsgProcessor
	lg   database.Login
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
	proc processors.MsgProcessor
	u    *model.Usr
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
	return u.proc.Process(&model.ReceivedMsg{
		From:    from,
		To:      to,
		Content: content,
	})
}

func (*sus) Logout() error {
	return nil
}
