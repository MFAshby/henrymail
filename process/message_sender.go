package process

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"henrymail/config"
	"net"
	"net/smtp"
	"strings"
)

/**
 * Forward messages on to their destinations by acting as an SMTP client.
 * Store messages and retry them later
 */
type sender struct {
	db *sql.DB
}

func (s *sender) Process(w *ReceivedMsg) error {
	errs := make([]string, 0)
	for _, to := range w.To {
		err := s.sendTo(to, w.From, w.Content)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	} else {
		return nil
	}
}

/**
 * Look for SMTP servers to send to,
 */
func (s *sender) sendTo(to, from string, content []byte) error {
	parts := strings.Split(to, "@")
	domain := parts[1]
	mxes, e := net.LookupMX(domain)
	if e != nil {
		return e
	}

	if len(mxes) == 0 {
		return errors.New("no MX records found for domain " + domain)
	}

	errs := make([]string, 0)
	for _, mx := range mxes {
		mailServer := strings.TrimRight(mx.Host, ".")
		e = s.sendToHost(to, from, mailServer, content)
		if e == nil {
			// If any server accepted the message then discard any errors,
			return nil
		} else {
			errs = append(errs, fmt.Sprintf("Host %v Error %v", mx.Host, e.Error()))
		}
	}
	return errors.New(strings.Join(errs, "\n"))
}

/**
 * Dials the other SMTP server and actually sends
 * the message. Error if anything went wrong.
 */
func (s *sender) sendToHost(to, from, host string, content []byte) error {
	client, err := smtp.Dial(host + config.GetString(config.MtaSendPort))
	if err != nil {
		return err
	}
	err = client.Hello(config.GetString(config.ServerName))
	if err != nil {
		return err
	}
	b, _ := client.Extension("STARTTLS")
	if b {
		err = client.StartTLS(&tls.Config{
			ServerName: host,
		})
	}
	if err != nil {
		return err
	}
	err = client.Mail(from)
	if err != nil {
		return err
	}
	err = client.Rcpt(to)
	if err != nil {
		return err
	}
	writeCloser, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writeCloser.Write(content)
	if err != nil {
		return err
	}
	err = writeCloser.Close()
	if err != nil {
		return err
	}
	return client.Quit()
}

func NewSender(db *sql.DB) *sender {
	sender := &sender{
		db: db,
	}
	return sender
}
