package main

import (
	"bytes"
	"github.com/emersion/go-dkim"
	"github.com/emersion/go-message"
	"time"
)

type HasId struct {
	Id int64
}

type Usr struct {
	HasId
	Email string
	Admin bool
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

func (m *Msg) Entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Content))
}

func (m *Msg) Subject() string {
	entity, e := m.Entity()
	if e != nil {
		return e.Error()
	}
	return entity.Header.Get("Subject")
}

func (m *Msg) From() string {
	entity, e := m.Entity()
	if e != nil {
		return e.Error()
	}
	return entity.Header.Get("From")
}

type MsgProcessor interface {
	Process(*ReceivedMsg) error
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
