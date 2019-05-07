package smtp

import (
	"bytes"
	"crypto/tls"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"henrymail/config"
	"henrymail/logic"
	"henrymail/process"
	"io"
	"io/ioutil"
	"log"
	"os"
)

/**
 * Accepts new mail from our own users for sending
 */
func StartMsa(proc process.MsgProcessor, tls *tls.Config) {
	be := &smtpSubmissionBackend{
		proc: proc,
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

type smtpSubmissionBackend struct {
	proc process.MsgProcessor
}

func (b *smtpSubmissionBackend) Login(username, password string) (smtp.User, error) {
	user, e := logic.Login(username, password)
	if e != nil {
		return nil, e
	}
	return &smtpUser{
		proc:   b.proc,
		userid: user.ID,
	}, nil
}

func (b *smtpSubmissionBackend) AnonymousLogin() (smtp.User, error) {
	return nil, smtp.ErrAuthRequired
}

type smtpUser struct {
	proc   process.MsgProcessor
	userid int
}

func (u *smtpUser) Send(from string, to []string, r io.Reader) error {
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
	return u.proc.Process(&process.ReceivedMsg{
		From:    from,
		To:      to,
		Content: content,
	})
}

func (*smtpUser) Logout() error {
	return nil
}
