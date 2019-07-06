package smtp

import (
	"bytes"
	"crypto/tls"
	"database/sql"
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
func StartMsa(db *sql.DB, proc process.MsgProcessor, tls *tls.Config) {
	be := &smtpSubmissionBackend{
		db:   db,
		proc: proc,
	}
	s := smtp.NewServer(be)
	s.Addr = config.GetString(config.MsaAddress)
	s.Domain = config.GetString(config.ServerName)
	//TODO come back to this
	// s.ReadTimeout = time.Duration config.GetInt(config.MaxIdleSeconds)
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
	db   *sql.DB
	proc process.MsgProcessor
}

func (b *smtpSubmissionBackend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	user, e := logic.Login(b.db, username, password)
	if e != nil {
		return nil, e
	}
	return &smtpSubmissionSession{
		proc:   b.proc,
		userid: user.ID,
	}, nil
}

func (b *smtpSubmissionBackend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

type smtpSubmissionSession struct {
	proc   process.MsgProcessor
	userid int
	currentFrom string
	currentTo []string
}

func (u *smtpSubmissionSession) Reset() {
	u.currentFrom = ""
	u.currentTo = make([]string, 0)
}

func (u *smtpSubmissionSession) Mail(from string) error {
	u.currentFrom = from
	return nil
}

func (u *smtpSubmissionSession) Rcpt(to string) error {
	u.currentTo = append(u.currentTo, to)
	return nil
}

func (u *smtpSubmissionSession) Data(r io.Reader) error {
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
		From:    u.currentFrom,
		To:      u.currentTo,
		Content: content,
	})
}

func (*smtpSubmissionSession) Logout() error {
	return nil
}
