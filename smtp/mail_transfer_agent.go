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
 * Accepts new mail from other servers
 */
func StartMta(proc processors.MsgProcessor, tls *tls.Config) {
	b := &tbe{
		proc: proc,
	}
	s := smtp.NewServer(b)
	s.Addr = config.GetString(config.MtaAddressKey)
	s.Domain = config.GetString(config.ServerNameKey)
	s.MaxIdleSeconds = config.GetInt(config.MaxIdleSecondsKey)
	s.MaxMessageBytes = config.GetInt(config.MaxMessageBytesKey)
	s.MaxRecipients = config.GetInt(config.MaxRecipientsKey)
	s.AuthDisabled = true
	s.Debug = os.Stdout
	s.TLSConfig = tls

	go func() {
		log.Println("Starting mail transfer agent at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type tbe struct {
	proc processors.MsgProcessor
	lg   database.Login
}

func (b *tbe) Login(username, password string) (smtp.User, error) {
	return nil, smtp.ErrAuthUnsupported
}

func (b *tbe) AnonymousLogin() (smtp.User, error) {
	return &tus{proc: b.proc}, nil
}

type tus struct {
	proc processors.MsgProcessor
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
	return u.proc.Process(&model.ReceivedMsg{
		From:    from,
		To:      to,
		Content: content,
	})
}

func (*tus) Logout() error {
	return nil
}
