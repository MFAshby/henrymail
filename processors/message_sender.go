package processors

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/emersion/go-message"
	"github.com/robfig/cron"
	"henrymail/config"
	"henrymail/database"
	"henrymail/model"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

/**
 * Forward messages on to their destinations by acting as an SMTP client.
 * Store messages and retry them later
 */
type sender struct {
	db database.Database
}

func (s *sender) Process(w *model.ReceivedMsg) error {
	errs := make([]string, 0)
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

func (s *sender) sendOrStoreForRetry(to, from string, timestamp time.Time, content []byte) error {
	if e := s.sendTo(to, from, content); e == nil {
		return nil
	}
	_, e := s.db.InsertQueue(from, to, content, timestamp)
	return e
}

/**
 * Look for SMTP servers to send to,
 */
func (s *sender) sendTo(to, from string, content []byte) error {
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
	return client.Quit()
}

func (s *sender) doRetries() {
	msgs, err := s.db.GetQueue()
	if err != nil {
		// Somehow we should alert, but smth is already wrong I guess
		log.Println(err)
		return
	}

	for _, q := range msgs {
		err := s.sendTo(q.To, q.From, q.Content)
		if err != nil {
			if q.Retries >= config.GetInt(config.RetryCount) {
				err = s.sendFailureNotification(q.To, q.From, q.Content, q.Retries)
				if err != nil {
					log.Println(err)
				}
				err = s.db.DeleteQueue(q.Id)
				if err != nil {
					log.Println(err)
				}
			} else {
				err = s.db.IncrementRetries(q.Id)
				if err != nil {
					log.Println(err)
				}
			}
		} else {
			err = s.db.DeleteQueue(q.Id)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (s *sender) sendFailureNotification(originalMsgTo, originalMsgFrom string, content []byte, retries int) error {
	// We will only be sending failure notifications to our own users.
	// so we can go direct to the database instead of talking to ourselves via SMTP.
	ibxId, e := s.db.GetInboxId(originalMsgFrom)
	if e != nil {
		return e
	}

	// render the error message template
	buf := new(bytes.Buffer)
	e = template.Must(template.ParseFiles("templates/failure_notification.content")).
		ExecuteTemplate(buf, "failure_notification.content", struct {
			To      string
			Retries int
		}{
			To:      originalMsgTo,
			Retries: retries,
		})
	if e != nil {
		return e
	}

	// Build a message from it
	retryPart, e := message.New(message.Header{}, buf)
	if e != nil {
		return e
	}

	// Read the original msg
	original, e := message.Read(bytes.NewReader(content))
	if e != nil {
		return e
	}

	// Bundle into a multipart message
	retryNotification, e := message.NewMultipart(message.Header{}, []*message.Entity{
		retryPart,
		original,
	})
	if e != nil {
		return e
	}

	// Save directly to the database
	buffer := new(bytes.Buffer)
	_ = retryNotification.WriteTo(buffer)
	_, e = s.db.InsertMessage(buffer.Bytes(), []string{}, ibxId, time.Now())
	return e
}

func NewSender(db database.Database) *sender {
	sender := &sender{
		db: db,
	}
	cr := cron.New()
	err := cr.AddFunc(config.GetString(config.RetryCronSpec), sender.doRetries)
	if err != nil {
		log.Fatal(err)
	}
	return sender
}
