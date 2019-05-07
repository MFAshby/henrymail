package process

import (
	"database/sql"
	"github.com/emersion/go-imap"
	"github.com/xo/xoutil"
	"henrymail/database"
	"henrymail/logic"
	"henrymail/models"
	"strings"
	"time"
)

/**
 * Saves which are intended for our own users into their inboxes.
 */
type saver struct{}

func (s *saver) Process(wrap *ReceivedMsg) error {
	return database.Transact(database.DB, func(tx *sql.Tx) error {
		for _, to := range wrap.To {
			username := strings.Split(to, "@")[0]
			user, e := models.UserByUsername(tx, username)
			if e != nil {
				return e
			}

			inbox, e := models.MailboxByUseridName(tx, user.ID, imap.InboxName)
			if e != nil {
				return e
			}

			e = logic.SaveMessages(tx, inbox, &models.Message{
				Ts:        xoutil.SqTime{Time: time.Now()},
				Flagsjson: []byte("[]"),
				Content:   wrap.Content,
			})
			if e != nil {
				return e
			}
		}
		return nil
	})
}

func NewSaver() MsgProcessor {
	return &saver{}
}
