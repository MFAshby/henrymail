package smtp

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"henrymail/config"
	"henrymail/process"
	"io"
	"io/ioutil"
	"log"
	"os"
)

/**
 * Accepts new mail from other servers
 */
func StartMta(db *sql.DB, proc process.MsgProcessor, tls *tls.Config) {
	b := &smtpTransferBackend{
		db:   db,
		proc: proc,
	}
	s := smtp.NewServer(b)
	s.Addr = config.GetString(config.MtaAddress)
	s.Domain = config.GetString(config.ServerName)
	// s.MaxIdleSeconds = config.GetInt(config.MaxIdleSeconds)
	s.MaxMessageBytes = config.GetInt(config.MaxMessageBytes)
	s.MaxRecipients = config.GetInt(config.MaxRecipients)
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

type smtpTransferBackend struct {
	db   *sql.DB
	proc process.MsgProcessor
}

func (b *smtpTransferBackend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return nil, smtp.ErrAuthUnsupported
}

func (b *smtpTransferBackend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &smtpSession{proc: b.proc}, nil
}

type smtpSession struct {
	proc process.MsgProcessor

	currentFrom string
	currentTo []string
}

func (s *smtpSession) Mail(from string, options smtp.MailOptions) error {
	s.currentFrom = from
	return nil
}

func (s *smtpSession) Rcpt(to string) error {
	s.currentTo = append(s.currentTo, to)
	return nil
}

func (s *smtpSession) Data(r io.Reader) error {
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
	return s.proc.Process(&process.ReceivedMsg{
		From:    s.currentFrom,
		To:      s.currentTo,
		Content: content,
	})
}

func (s *smtpSession) Reset() {
	s.currentFrom = ""
	s.currentTo = make([]string, 0)
}

func (*smtpSession) Logout() error {
	return nil
}