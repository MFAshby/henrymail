package logic

import (
	"database/sql"
	"github.com/emersion/go-imap"
	"henrymail/models"
	"strings"
)

/**
 * Common functions for mail logic
 */

/**
 * We do this in a few places, might make it a custom query
 */
func FindInbox(db models.XODB, emailaddress string) (*models.Mailbox, error) {
	split := strings.Split(emailaddress, "@")
	username := split[0]
	user, e := models.UserByUsername(db, username)
	if e != nil {
		return nil, e
	}
	return models.MailboxByUseridName(db, user.ID, imap.InboxName)
}

/**
 * Should be done in a transaction since multiple updates are required
 */
func SaveMessages(tx *sql.Tx, mailbox *models.Mailbox, messages ...*models.Message) error {
	for _, msg := range messages {
		// Ensure the link
		msg.Mailboxid = mailbox.ID
		// And the UID
		msg.UID = mailbox.Uidnext
		// increment the UID
		mailbox.Uidnext += 1

		e := msg.Save(tx)
		if e != nil {
			return e
		}
	}
	// Ensure the new Uidnext is saved on the mailbox too
	return mailbox.Save(tx)
}
