package model

import (
	"bytes"
	"errors"
	"github.com/emersion/go-dkim"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-message"
	"html/template"
	"io"
	"io/ioutil"
	"time"
)

type HasId struct {
	Id int64
}

type Usr struct {
	HasId
	Username string
	Admin    bool
}

type Mbx struct {
	HasId
	UserId      int64
	Name        string
	Messages    uint32
	Recent      uint32
	Unseen      uint32
	UidNext     uint32
	UidValidity uint32
}

type Msg struct {
	HasId
	MbxId     int64
	Content   []byte
	Uid       uint32
	Flags     []string
	Timestamp time.Time
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

func (m *Msg) Entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Content))
}

func (m *Msg) headerOrError(header string) string {
	entity, e := m.Entity()
	if e != nil {
		return e.Error()
	}
	return entity.Header.Get(header)
}

func (m *Msg) Subject() string {
	return m.headerOrError("Subject")
}

func (m *Msg) From() string {
	return m.headerOrError("From")
}

func (m *Msg) To() string {
	return m.headerOrError("To")
}

func (m *Msg) PlainBody() string {
	entity, e := m.Entity()
	if e != nil {
		return e.Error()
	}
	content, e := getContent(entity, "text/plain")
	if e != nil {
		return e.Error()
	} else {
		return content
	}
}

func (m *Msg) HtmlBody() template.HTML {
	entity, e := m.Entity()
	if e != nil {
		return template.HTML(e.Error())
	}
	content, e := getContent(entity, "text/html")
	if e != nil {
		return template.HTML(e.Error())
	} else {
		return template.HTML(content)
	}
}

func getContent(ent *message.Entity, contentType string) (string, error) {
	mpr := ent.MultipartReader()
	if mpr != nil {
		for ent2, e := mpr.NextPart(); e != io.EOF; {
			c, e := getContentNoMultipart(ent2, contentType)
			if e == nil {
				return c, nil
			}
		}
	}
	return getContentNoMultipart(ent, contentType)
}

func getContentNoMultipart(ent *message.Entity, contentType string) (string, error) {
	t, _, e := ent.Header.ContentType()
	if e != nil {
		return "", e
	}
	if t == contentType {
		all, e := ioutil.ReadAll(ent.Body)
		if e != nil {
			return "", e
		}
		return string(all), nil
	}
	return "", errors.New("wrong content type " + contentType)
}

type ReceivedMsg struct {
	From      string
	To        []string
	Content   []byte
	Timestamp time.Time

	Verifications []*dkim.Verification
}

type QueuedMsg struct {
	HasId
	From      string
	To        string
	Content   []byte
	Timestamp time.Time
	Retries   int
}
