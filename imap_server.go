package main

import (
	"bytes"
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-imap/server"
	message2 "github.com/emersion/go-message"
	"io/ioutil"
	"log"
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
			x:  mbx,
			db: u.db,
		}
	}
	return mailboxes, nil
}

func (u *ius) GetMailbox(name string) (backend.Mailbox, error) {
	mbx, e := u.db.GetMailbox(name, u.user.Id)
	if e != nil {
		return nil, e
	}
	return &imb{
		x:  mbx,
		db: u.db,
	}, nil
}

func (u *ius) CreateMailbox(name string) error {
	_, e := u.db.InsertMailbox(name, u.user.Id)
	return e
}

func (u *ius) DeleteMailbox(name string) error {
	return errors.New("operation not supported yet")
}

func (u *ius) RenameMailbox(existingName, newName string) error {
	return errors.New("operation not supported yet")
}

func (*ius) Logout() error {
	return nil
}

type imb struct {
	x  *Mbx
	db Database
}

func (m *imb) Name() string {
	return m.x.Name
}

func (m *imb) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: []string{},
		Delimiter:  "/",
		Name:       m.Name(),
	}, nil
}

func (m *imb) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	status := imap.NewMailboxStatus(m.x.Name, items)
	//status.Flags = m.x.Flags
	status.PermanentFlags = []string{"\\*"}
	//status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = m.x.Messages
		case imap.StatusUidNext:
			status.UidNext = m.x.UidNext
		case imap.StatusUidValidity:
			status.UidValidity = m.x.UidValidity
		case imap.StatusRecent:
			status.Recent = m.x.Recent
		case imap.StatusUnseen:
			status.Unseen = m.x.Unseen
		}
	}

	return status, nil
}

func (m *imb) SetSubscribed(subscribed bool) error {
	return errors.New("operation not supported")
}

func (*imb) Check() error {
	return nil
}

func (m *imb) ListMessages(uid bool, seqset *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)
	if !uid {
		return errors.New("operation not yet supported")
	}

	// Two passes, because seqset may require multiple database fetches
	messages := make([]*Msg, 0)
	for _, seq := range seqset.Set {
		// Unbounded search on UID
		upperUid := -1
		if seq.Stop > 0 {
			upperUid = int(seq.Stop)
		}
		msgs, e := m.db.GetMessages(m.x.Id, int(seq.Start), upperUid)
		if e != nil {
			return e
		}
		messages = append(messages, msgs...)
	}

	for ix, msg := range messages {
		message, e := msg.Fetch(uint32(ix), items)
		if e != nil {
			return e
		}
		ch <- message
	}
	return nil
}

func (m *Msg) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.FetchEnvelope:
			e, _ := message2.Read(bytes.NewReader(m.Content))
			fetched.Envelope, _ = backendutil.FetchEnvelope(e.Header)
		case imap.FetchBody, imap.FetchBodyStructure:
			e, _ := message2.Read(bytes.NewReader(m.Content))
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

			e, _ := message2.Read(bytes.NewReader(m.Content))
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
	//if !backendutil.MatchDate(m.Date, c) {
	//	return false, errors.New("operation not supported")
	//}
	if !backendutil.MatchFlags(m.Flags, c) {
		return false, nil
	}

	e, _ := m.Entity()
	return backendutil.Match(e, c)
}

func (m *imb) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	if !uid {
		return nil, errors.New("uid not supported in SearchMessages")
	}

	msgs, e := m.db.GetMessages(m.x.Id, -1, -1)
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
	content, e := ioutil.ReadAll(body)
	if e != nil {
		return e
	}
	_, e = m.db.InsertMessage(content, flags, m.x.Id)
	return e
}

func (*imb) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	return errors.New("operation not supported")
}

func (*imb) CopyMessages(uid bool, seqset *imap.SeqSet, dest string) error {
	return errors.New("operation not supported")
}

func (*imb) Expunge() error {
	return errors.New("operation not supported")
}

func StartImap(lg Login, db Database) {
	be := &ibe{
		lg: lg,
		db: db,
	}
	s := server.New(be)
	s.Addr = GetString(ImapAddressKey)
	s.AllowInsecureAuth = true
	go func() {
		log.Println("Starting IMAP server at ", s.Addr)
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}
