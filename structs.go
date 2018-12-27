package main

import (
	"bytes"
	"github.com/emersion/go-message"
	"time"
)

type MsgProcessor interface {
	Process(*Wrap) error
}

type Wrap struct {
	From      string
	To        []string
	Content   []byte
	Timestamp time.Time
}

type QueuedMsg struct {
	From      string
	To        string
	Content   []byte
	Timestamp time.Time
	retries   uint32
}

type Usr struct {
	Id    int64
	Email string
}

type Mbx struct {
	Id          int64
	UserId      int64
	Name        string
	Messages    uint32
	Recent      uint32
	Unseen      uint32
	UidNext     uint32
	UidValidity uint32
}

type Msg struct {
	Id      int64
	MbxId   int64
	Content []byte
	Uid     uint32
	Flags   []string
}

func (m *Msg) Entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Content))
}
