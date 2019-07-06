package imap

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-message"
	"github.com/xo/xoutil"
	"henrymail/config"
	"henrymail/database"
	"henrymail/logic"
	"henrymail/models"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func StartImap(db *sql.DB, tls *tls.Config) {
	be := &imapBackend{
		db: db,
	}
	s := server.New(be)
	s.Addr = config.GetString(config.ImapAddress)
	s.Debug = os.Stdout
	s.TLSConfig = tls
	s.AllowInsecureAuth = !config.GetBool(config.ImapUseTls)
	go func() {
		log.Println("Starting IMAP server at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

type imapBackend struct {
	db *sql.DB
}

func (b *imapBackend) Login(connInfo *imap.ConnInfo, username, password string) (backend.User, error) {
	user, e := logic.Login(b.db, username, password)
	if e != nil {
		return nil, e
	}
	return &imapUser{
		userid: user.ID,
		db:     b.db,
	}, nil
}

type imapUser struct {
	userid int
	db     *sql.DB
}

func (u *imapUser) Username() string {
	user, e := models.UserByID(u.db, u.userid)
	if e != nil {
		return "UNKNOWN"
	}
	return user.Username
}

func (u *imapUser) ListMailboxes(subscribed bool) ([]backend.Mailbox, error) {
	mbxs, err := models.MailboxesByUserid(u.db, u.userid)
	if err != nil {
		return nil, err
	}
	mailboxes := make([]backend.Mailbox, len(mbxs))
	for ix, mbx := range mbxs {
		mailboxes[ix] = &imapMailbox{
			mailboxid: mbx.ID,
			db:        u.db,
		}
	}
	return mailboxes, nil
}

func (u *imapUser) GetMailbox(name string) (backend.Mailbox, error) {
	mailbox, e := models.MailboxByUseridName(u.db, u.userid, name)
	if e != nil {
		return nil, e
	}
	return &imapMailbox{
		mailboxid: mailbox.ID,
		db:        u.db,
		userid:    u.userid,
	}, nil
}

func (u *imapUser) CreateMailbox(name string) error {
	mailbox := &models.Mailbox{
		Name:       name,
		Userid:     u.userid,
		Subscribed: true,
	}
	return mailbox.Save(u.db)
}

func (u *imapUser) DeleteMailbox(name string) error {
	mailbox, e := models.MailboxByUseridName(u.db, u.userid, name)
	if e != nil {
		return e
	}
	return mailbox.Delete(u.db)
}

func (u *imapUser) RenameMailbox(existingName, newName string) error {
	mailbox, e := models.MailboxByUseridName(u.db, u.userid, existingName)
	if e != nil {
		return e
	}
	mailbox.Name = newName
	return mailbox.Save(u.db)
}

func (*imapUser) Logout() error {
	return nil
}

// Only store the ID, so we dont end up with stale data being read!
type imapMailbox struct {
	userid    int
	mailboxid int
	db        *sql.DB
}

func (m *imapMailbox) Name() string {
	mailbox, e := models.MailboxByID(m.db, m.mailboxid)
	if e != nil {
		return "UNKNOWN"
	}
	return mailbox.Name
}

func (m *imapMailbox) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: []string{},
		Delimiter:  "/",
		Name:       m.Name(),
	}, nil
}

func (m *imapMailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	mbx, err := models.MailboxByID(m.db, m.mailboxid)
	if err != nil {
		return nil, err
	}
	//TODO make this more efficient
	messages, err := models.MessagesByMailboxid(m.db, m.mailboxid)
	if err != nil {
		return nil, err
	}

	status := imap.NewMailboxStatus(mbx.Name, items)

	//TODO fill this in correctly
	//status.Flags = m.x.Flags
	status.PermanentFlags = []string{"\\*"}
	//status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(messages))
		case imap.StatusUidNext:
			status.UidNext = uint32(mbx.Uidnext)
		case imap.StatusUidValidity:
			status.UidValidity = uint32(mbx.Uidvalidity)
		// TODO fill these in correctly
		case imap.StatusRecent:
			status.Recent = 0
		case imap.StatusUnseen:
			status.Unseen = 0
		}
	}

	return status, nil
}

func (m *imapMailbox) SetSubscribed(subscribed bool) error {
	mailbox, e := models.MailboxByID(m.db, m.mailboxid)
	if e != nil {
		return e
	}
	mailbox.Subscribed = subscribed
	return mailbox.Save(m.db)
}

func (*imapMailbox) Check() error {
	return nil
}

func (m *imapMailbox) ListMessages(uid bool, seqset *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)
	messages, e := models.MessagesByMailboxid(m.db, m.mailboxid)
	if e != nil {
		return e
	}
	return applyToSet(messages, uid, seqset, func(msg *models.Message, seqnum uint32) error {
		imsg, e := fetch(msg, seqnum, items)
		if e != nil {
			return e
		}
		ch <- imsg
		return nil
	})
}

func (m *imapMailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	msgs, e := models.MessagesByMailboxid(m.db, m.mailboxid)
	if e != nil {
		return nil, e
	}
	var matches []uint32
	for ix, msg := range msgs {
		seqnum := uint32(ix + 1)
		match, e := match(msg, seqnum, criteria)
		if e != nil {
			return nil, e
		}
		if match {
			if uid {
				matches = append(matches, uint32(msg.UID))
			} else {
				matches = append(matches, seqnum)
			}
		}
	}
	return matches, nil
}

func (m *imapMailbox) CreateMessage(flags []string, ts time.Time, body imap.Literal) error {
	return database.Transact(m.db, func(tx *sql.Tx) error {
		mailbox, e := models.MailboxByID(tx, m.mailboxid)
		if e != nil {
			return e
		}
		content, e := ioutil.ReadAll(body)
		if e != nil {
			return e
		}

		return logic.SaveMessages(tx, mailbox, &models.Message{
			Mailboxid: mailbox.ID,
			UID:       mailbox.Uidnext,
			Flagsjson: []byte("[]"),
			Content:   content,
			Ts:        xoutil.SqTime{Time: ts},
		})
	})
}

func (m *imapMailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	return database.Transact(m.db, func(tx *sql.Tx) error {
		messages, e := models.MessagesByMailboxid(tx, m.mailboxid)
		if e != nil {
			return e
		}

		return applyToSet(messages, uid, seqset, func(msg *models.Message, _ uint32) error {
			existingFlags, e := getFlags(msg)
			if e != nil {
				return e
			}

			var newFlags []string
			switch operation {
			case imap.SetFlags:
				newFlags = flags
			case imap.AddFlags:
				newFlags = append(existingFlags, flags...)
			case imap.RemoveFlags:
				for _, existingFlag := range existingFlags {
					if !stringSl(flags).contains(existingFlag) {
						newFlags = append(newFlags, existingFlag)
					}
				}
			default:
				return errors.New(fmt.Sprintf("unexpected flags operation %v", operation))
			}

			e = setFlags(msg, newFlags)
			if e != nil {
				return e
			}
			return msg.Save(tx)
		})
	})
}

func (m *imapMailbox) CopyMessages(uid bool, seqset *imap.SeqSet, dest string) error {
	return database.Transact(m.db, func(tx *sql.Tx) error {
		messages, e := models.MessagesByMailboxid(tx, m.mailboxid)
		if e != nil {
			return e
		}
		destmailbox, e := models.MailboxByUseridName(tx, m.userid, dest)
		if e != nil {
			return e
		}
		var newMessages []*models.Message
		e = applyToSet(messages, uid, seqset, func(msg *models.Message, _ uint32) error {
			newMessages = append(newMessages, &models.Message{
				Ts:        msg.Ts,
				Content:   msg.Content,
				Flagsjson: msg.Flagsjson,
			})
			return nil
		})
		return logic.SaveMessages(tx, destmailbox, newMessages...)
	})
}

func (m *imapMailbox) Expunge() error {
	return database.Transact(m.db, func(tx *sql.Tx) error {
		messages, e := models.MessagesByMailboxid(m.db, m.mailboxid)
		if e != nil {
			return e
		}

		for _, msg := range messages {
			flags, e := getFlags(msg)
			if stringSl(flags).contains(imap.DeletedFlag) {
				e = msg.Delete(tx)
				if e != nil {
					return e
				}
			}
		}
		return nil
	})
}

func applyToSet(messages []*models.Message, uid bool, set *imap.SeqSet, f func(*models.Message, uint32) error) error {
	for ix, msg := range messages {
		seqnum := uint32(ix + 1)
		var check uint32
		if uid {
			check = uint32(msg.UID)
		} else {
			check = seqnum
		}
		if set.Contains(check) {
			e := f(msg, seqnum)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

type stringSl []string

func (sl stringSl) contains(s string) bool {
	for _, x := range sl {
		if s == x {
			return true
		}
	}
	return false
}

func getFlags(m *models.Message) ([]string, error) {
	var flags []string
	e := json.Unmarshal(m.Flagsjson, &flags)
	return flags, e
}

func setFlags(m *models.Message, flags []string) error {
	flagsJson, e := json.Marshal(flags)
	if e != nil {
		return e
	}
	m.Flagsjson = flagsJson
	return nil
}

func match(m *models.Message, seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	msgflags, e := getFlags(m)
	if e != nil {
		return false, e
	}
	ent, e := entity(m)
	if e != nil {
		return false, e
	}
	return backendutil.Match(ent, seqNum, uint32(m.UID), m.Ts.Time, msgflags, c)
}

func fetch(m *models.Message, seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.FetchEnvelope:
			ent, e := message.Read(bytes.NewReader(m.Content))
			if e != nil {
				return nil, e
			}
			fetched.Envelope, _ = backendutil.FetchEnvelope(ent.Header.Header)
		case imap.FetchBody, imap.FetchBodyStructure:
			ent, e := message.Read(bytes.NewReader(m.Content))
			if e != nil {
				return nil, e
			}
			fetched.BodyStructure, _ = backendutil.FetchBodyStructure(ent.Header.Header, ent.Body, item == imap.FetchBodyStructure)
		case imap.FetchFlags:
			flags, e := getFlags(m)
			if e != nil {
				return nil, e
			}
			fetched.Flags = flags
		case imap.FetchInternalDate:
			//TODO what is this
			fetched.InternalDate = time.Now()
		case imap.FetchRFC822Size:
			fetched.Size = uint32(len(m.Content))
		case imap.FetchUid:
			fetched.Uid = uint32(m.UID)
		default:
			section, err := imap.ParseBodySectionName(item)
			if err != nil {
				break
			}

			e, _ := message.Read(bytes.NewReader(m.Content))
			l, _ := backendutil.FetchBodySection(e.Header.Header, e.Body, section)
			fetched.Body[section] = l
		}
	}

	return fetched, nil
}

func entity(m *models.Message) (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Content))
}
