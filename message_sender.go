package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/robfig/cron"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

/**
 * Forward messages on to their destinations by acting as an SMTP client.
 */
type Sender struct {
	db Database
}

func (s *Sender) Process(w *Wrap) error {
	errs := make([]string, 0)
	// Split out and call each of the recipients SMTP servers individually.
	// This could be more efficient, grouping by host etc but it's not for now.
	for _, to := range w.To {
		err := s.sendOrStoreForRetry(to, w.From, w.Timestamp, w.Content)
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

func (s *Sender) StartRetries() error {
	cr := cron.New()
	return cr.AddFunc(GetString(RetryCronSpec), func() {
		msgs, err := s.db.GetQueuedMsgs()
		if err != nil {
			// Somehow we should alert, but smth is already wrong I guess
			log.Println(err)
			return
		}

		for _, q := range msgs {
			err := s.sendTo(q.To, q.From, q.Content)
			if err != nil {
				// Increment & check retry limit,
				// Msg back to sender
				// Delete from queue
			} else {
				// Delete from queue
			}
		}
	})
}

func (s *Sender) sendOrStoreForRetry(to, from string, timestamp time.Time, content []byte) error {
	if e := s.sendTo(to, from, content); e == nil {
		return nil
	}
	return s.db.InsertQueue(from, to, content, timestamp)
}

/**
 * Look for SMTP servers to send to,
 */
func (s *Sender) sendTo(to, from string, content []byte) error {
	parts := strings.Split(to, "@")
	host := parts[1]
	mxes, e := net.LookupMX(host)
	if e != nil {
		return e
	}

	if len(mxes) == 0 {
		return errors.New("no MX records found for host " + host)
	}

	errs := make([]string, 0)
	for _, mx := range mxes {
		e = s.sendToHost(to, from, mx.Host, content)
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
func (s *Sender) sendToHost(to, from, host string, content []byte) error {
	client, err := smtp.Dial(host + ":25")
	if err != nil {
		return err
	}
	err = client.Hello(GetString(DomainKey))
	if err != nil {
		return err
	}
	err = client.StartTLS(&tls.Config{
		ServerName: host,
	})
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
	return client.Quit()
}

func NewSender(db Database) *Sender {
	return &Sender{
		db: db,
	}
}
