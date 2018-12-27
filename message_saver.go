package main

import (
	"errors"
	"strings"
	"time"
)

/**
 * Saves which are intended for our own users into their inboxes.
 */
type saver struct {
	db Database
}

func (s *saver) Process(wrap *ReceivedMsg) error {
	// Find all the inboxes for each user, if we can't do them all we'll do none.
	inboxIds := make(map[string]int64)
	for _, to := range wrap.To {
		ibxId, e := s.db.GetInboxId(to)
		if e != nil {
			return NotFound
		}
		inboxIds[to] = ibxId
	}

	// Save the messages, we'll try to do them all.
	//  Even if we fail for some reason we'll try the others.
	errs := make([]string, 0)
	for _, to := range wrap.To {
		ibxId := inboxIds[to]
		_, e := s.db.InsertMessage(wrap.Content, []string{}, ibxId, time.Now())
		if e != nil {
			errs = append(errs, e.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ","))
	} else {
		return nil
	}
}

func NewSaver(db Database) MsgProcessor {
	return &saver{db: db}
}
