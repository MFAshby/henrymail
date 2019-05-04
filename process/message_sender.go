package process

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"github.com/emersion/go-message"
	"github.com/robfig/cron"
	"github.com/xo/xoutil"
	"henrymail/config"
	"henrymail/database"
	"henrymail/logic"
	"henrymail/models"
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
	db *sql.DB
	cr *cron.Cron
}

func (s *sender) Process(w *ReceivedMsg) error {
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
	//TODO check the severity of the error
	// Should check if the error is fatal or not really
	q := &models.Queue{
		Msgto:   to,
		Msgfrom: from,
		Ts:      xoutil.SqTime{Time: timestamp},
		Retries: 0,
		Content: content,
	}
	return q.Save(s.db)
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
	return client.Quit()
}

func (s *sender) doRetries() {
	queue, err := models.GetAllQueue(s.db)
	if err != nil {
		//TODO alert for asynchronous errors of some kind (email to admin?)
		log.Println(err)
		return
	}

	for _, q := range queue {
		err := s.sendTo(q.Msgto, q.Msgfrom, q.Content)
		//TODO refine & document conditions for retries
		if err != nil {
			if q.Retries >= config.GetInt(config.RetryCount) {
				err = s.sendFailureNotification(q.Msgto, q.Msgfrom, q.Content, q.Retries)
				if err != nil {
					log.Println(err)
				}
				err = q.Delete(s.db)
				if err != nil {
					log.Println(err)
				}
			} else {
				q.Retries += 1
				err = q.Save(s.db)
				if err != nil {
					log.Println(err)
				}
			}
		} else {
			err = q.Delete(s.db)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (s *sender) sendFailureNotification(originalMsgTo, originalMsgFrom string, content []byte, retries int) error {
	// We will only be sending failure notifications to our own users.
	inbox, e := logic.FindInbox(s.db, originalMsgFrom)
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

	buffer := new(bytes.Buffer)
	e = retryNotification.WriteTo(buffer)
	if e != nil {
		return e
	}

	return database.Transact(s.db, func(tx *sql.Tx) error {
		return logic.SaveMessages(tx, inbox, &models.Message{
			Content:   buffer.Bytes(),
			Flagsjson: []byte("[]"),
			Ts:        xoutil.SqTime{Time: time.Now()},
		})
	})
}

func NewSender(db *sql.DB) *sender {
	sender := &sender{
		db: db,
		cr: cron.New(),
	}
	err := sender.cr.AddFunc(config.GetString(config.RetryCronSpec), sender.doRetries)
	if err != nil {
		log.Fatal(err)
	}
	return sender
}
