package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-message"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type ibe struct {
	lg Login
	db Database
}

func (b *ibe) Login(username, password string) (backend.User, error) {
	user, e := b.lg.Login(username, password)
	if e != nil {
		return nil, e
	}
	return &ius{
		user: user,
		db:   b.db,
	}, nil
}

type ius struct {
	user *Usr
	db   Database
}

func (u *ius) Username() string {
	return u.user.Email
}

func (u *ius) ListMailboxes(subscribed bool) ([]backend.Mailbox, error) {
	mbxs, err := u.db.GetMailboxes(subscribed, u.user.Id)
	if err != nil {
		return nil, err
	}
	mailboxes := make([]backend.Mailbox, len(mbxs))
	for ix, mbx := range mbxs {
		mailboxes[ix] = &imb{
			mbxId: mbx.Id,
			db:    u.db,
		}
	}
	return mailboxes, nil
}

func (u *ius) GetMailbox(name string) (backend.Mailbox, error) {
	mbx, e := u.db.GetMailboxByName(name, u.user.Id)
	if e != nil {
		return nil, e
	}
	return &imb{
		mbxId: mbx.Id,
		db:    u.db,
	}, nil
}

func (u *ius) CreateMailbox(name string) error {
	_, e := u.db.InsertMailbox(name, u.user.Id)
	return e
}

func (u *ius) DeleteMailbox(name string) error {
	return u.db.DeleteMailbox(name, u.user.Id)
}

func (u *ius) RenameMailbox(existingName, newName string) error {
	return u.db.RenameMailbox(u.user.Id, existingName, newName)
}

func (*ius) Logout() error {
	return nil
}

// Only store the ID, so we dont end up with stale data being read!
type imb struct {
	mbxId int64
	db    Database
}

func (m *imb) getMbx() *Mbx {
	mbx, e := m.db.GetMailboxById(m.mbxId)
	if e != nil {
		log.Println(e)
		return &Mbx{}
	}
	return mbx
}

func (m *imb) Name() string {
	mbx, e := m.db.GetMailboxById(m.mbxId)
	if e != nil {
		return "ERROR"
	}
	return mbx.Name
}

func (m *imb) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: []string{},
		Delimiter:  "/",
		Name:       m.Name(),
	}, nil
}

func (m *imb) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	mbx := m.getMbx()
	status := imap.NewMailboxStatus(mbx.Name, items)
	//status.Flags = m.x.Flags
	status.PermanentFlags = []string{"\\*"}
	//status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = mbx.Messages
		case imap.StatusUidNext:
			status.UidNext = mbx.UidNext
		case imap.StatusUidValidity:
			status.UidValidity = mbx.UidValidity
		case imap.StatusRecent:
			status.Recent = mbx.Recent
		case imap.StatusUnseen:
			status.Unseen = mbx.Unseen
		}
	}

	return status, nil
}

func (m *imb) SetSubscribed(subscribed bool) error {
	return m.db.SetMailboxSubscribed(m.mbxId, subscribed)
}

func (*imb) Check() error {
	return nil
}

func (m *imb) ListMessages(uid bool, seqset *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)
	msgs, e := m.getAllMessages()
	if e != nil {
		return e
	}
	for ix, msg := range msgs {
		var check uint32
		if uid {
			check = msg.Uid
		} else {
			check = uint32(ix)
		}
		if seqset.Contains(check) {
			// Seqnum is an index from 1 :(
			imsg, e := msg.Fetch(uint32(ix+1), items)
			if e != nil {
				return e
			}
			ch <- imsg
		}
	}
	return nil
}

func (m *imb) getAllMessages() ([]*Msg, error) {
	return m.db.GetMessages(m.mbxId, -1, -1)
}

func (m *Msg) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.FetchEnvelope:
			e, _ := message.Read(bytes.NewReader(m.Content))
			fetched.Envelope, _ = backendutil.FetchEnvelope(e.Header)
		case imap.FetchBody, imap.FetchBodyStructure:
			e, _ := message.Read(bytes.NewReader(m.Content))
			fetched.BodyStructure, _ = backendutil.FetchBodyStructure(e, item == imap.FetchBodyStructure)
		case imap.FetchFlags:
			fetched.Flags = m.Flags
		case imap.FetchInternalDate:
			fetched.InternalDate = time.Now()
		case imap.FetchRFC822Size:
			fetched.Size = uint32(len(m.Content))
		case imap.FetchUid:
			fetched.Uid = m.Uid
		default:
			section, err := imap.ParseBodySectionName(item)
			if err != nil {
				break
			}

			e, _ := message.Read(bytes.NewReader(m.Content))
			l, _ := backendutil.FetchBodySection(e, section)
			fetched.Body[section] = l
		}
	}

	return fetched, nil
}

func (m *Msg) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	if !backendutil.MatchSeqNumAndUid(seqNum, m.Uid, c) {
		return false, nil
	}
	if !backendutil.MatchDate(m.Timestamp, c) {
		return false, nil
	}
	if !backendutil.MatchFlags(m.Flags, c) {
		return false, nil
	}

	e, _ := m.Entity()
	return backendutil.Match(e, c)
}

func (m *Msg) Entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Content))
}

func (m *imb) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	if !uid {
		return nil, errors.New("non-uid not supported in SearchMessages")
	}
	msgs, e := m.getAllMessages()
	if e != nil {
		return nil, e
	}
	var matches []uint32
	for ix, msg := range msgs {
		b, e := msg.Match(uint32(ix), criteria)
		if e != nil {
			return nil, e
		}
		if b {
			matches = append(matches, msg.Uid)
		}
	}
	return matches, nil
}

func (m *imb) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	mbx := m.getMbx()
	content, e := ioutil.ReadAll(body)
	if e != nil {
		return e
	}
	_, e = m.db.InsertMessage(content, flags, mbx.Id, time.Now())
	return e
}

func (m *imb) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	if !uid {
		return errors.New("operation not supported")
	}
	msgs, e := m.getAllMessages()
	if e != nil {
		return e
	}

	for ix, msg := range msgs {
		var check uint32
		if uid {
			check = msg.Uid
		} else {
			check = uint32(ix)
		}
		if seqset.Contains(check) {
			var newFlags []string
			switch operation {
			case imap.SetFlags:
				newFlags = flags
			case imap.AddFlags:
				newFlags = append(msg.Flags, flags...)
			case imap.RemoveFlags:
				newFlags := msg.Flags[:0]
				var f stringSl = flags
				for _, x := range msg.Flags {
					if !f.contains(x) {
						newFlags = append(newFlags, x)
					}
				}
			default:
				return errors.New(fmt.Sprintf("unexpected flags operation %v", operation))
			}
			e := m.db.SetMessageFlags(msg.Id, newFlags)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (m *imb) CopyMessages(uid bool, seqset *imap.SeqSet, dest string) error {
	if !uid {
		return errors.New("operation not supported")
	}
	mbx := m.getMbx()
	destMbx, e := m.db.GetMailboxByName(dest, mbx.UserId)
	if e != nil {
		return e
	}
	for _, seq := range seqset.Set {
		stop := -1
		if seq.Stop > 0 {
			stop = int(seq.Stop)
		}
		msgs, e := m.db.GetMessages(m.mbxId, int(seq.Start), stop)
		if e != nil {
			return e
		}
		for _, msg := range msgs {
			_, e := m.db.InsertMessage(msg.Content, msg.Flags, destMbx.Id, time.Now())
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (m *imb) Expunge() error {
	msgs, e := m.db.GetMessages(m.mbxId, -1, -1)
	if e != nil {
		return e
	}

	for _, msg := range msgs {
		var f stringSl = msg.Flags
		if f.contains(imap.DeletedFlag) {
			e = m.db.DeleteMessage(msg.Id)
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

func StartImap(lg Login, db Database, config *tls.Config) {
	be := &ibe{
		lg: lg,
		db: db,
	}
	s := server.New(be)
	s.Addr = GetString(ImapAddressKey)
	s.Debug = os.Stdout
	s.TLSConfig = config
	go func() {
		log.Println("Starting IMAP server at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}
