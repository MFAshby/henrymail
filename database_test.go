package main

import (
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"testing"
	"time"
)

var date = time.Date(2018, 12, 1, 0, 0, 0, 0, time.UTC)

func withDb(t *testing.T, f func(t *testing.T, d Database)) {
	viper.Set(DbConnectionStringKey, "file::memory:?mode=memory&cache=shared")
	viper.Set(DbDriverNameKey, "sqlite3")
	database := NewDatabase()
	f(t, database)
}

func TestUserCrud(t *testing.T) {
	withDb(t, func(t *testing.T, d Database) {
		email := "hello@blah.com"
		// CREATE
		usr, e := d.InsertUser(email, []byte("testing"), false)
		if e != nil {
			t.Error(e)
		}

		// READ
		usr2, _, e := d.GetUserAndPassword(email)
		if diff := cmp.Diff(usr, usr2); diff != "" {
			t.Errorf("User different after GetUserAndPassword %s", diff)
		}

		usrs, e := d.GetUsers()
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(usrs, []*Usr{usr}); diff != "" {
			t.Errorf("User different after GetUsers %s", diff)
		}

		// DELETE
		e = d.DeleteUser(email)
		if e != nil {
			t.Error(e)
		}
		_, _, e = d.GetUserAndPassword(email)
		if e == nil {
			t.Error("Expected error retrieving after deletion")
		}
	})
}

func TestMessageCrud(t *testing.T) {
	withDb(t, func(t *testing.T, d Database) {
		// CREATE
		_, e := d.InsertMessage([]byte("blah"), []string{"one", "two"}, -1, date)
		if e == nil {
			t.Errorf("Expected error inserting message with no mailbox")
		}
		u, _ := d.InsertUser("test@blah.com", []byte("blah"), false)
		mbx, _ := d.InsertMailbox("ibx", u.Id)
		msg, e := d.InsertMessage([]byte("blah"), []string{"one", "two"}, mbx.Id, date)
		if e != nil {
			t.Error(e)
		}
		if msg.Id == 0 {
			t.Error("msg.Id should have been set")
		}

		// READ
		msgs, e := d.GetMessages(mbx.Id, -1, -1)
		if e != nil {
			t.Error(e)
		}
		if dif := cmp.Diff(msgs, []*Msg{msg}); dif != "" {
			t.Errorf("GetMessages comparison failed: %s", dif)
		}

		// UPDATE
		e = d.SetMessageFlags(msg.Id, []string{"three"})
		if e != nil {
			t.Error(e)
		}
		msgs, _ = d.GetMessages(mbx.Id, -1, -1)
		msg.Flags = []string{"three"}
		if dif := cmp.Diff(msgs, []*Msg{msg}); dif != "" {
			t.Errorf("GetMessages comparison failed after updating flags: %s", dif)
		}

		// DELETE
		e = d.DeleteMessage(msg.Id)
		if e != nil {
			t.Error(e)
		}

		msgs, e = d.GetMessages(mbx.Id, -1, -1)
		if e != nil {
			t.Error(e)
		}
		if dif := cmp.Diff(msgs, []*Msg{}); dif != "" {
			t.Errorf("GetMessages comparison failed after deleting: %s", dif)
		}
	})
}

func TestMailboxCrud(t *testing.T) {
	withDb(t, func(t *testing.T, d Database) {
		// CREATE
		_, e := d.InsertMailbox("blah", -1)
		if e == nil {
			t.Errorf("Expecting error inserting mailbox with no user")
		}

		usr, _ := d.InsertUser("blah@test.com", []byte("something"), false)
		mbx, e := d.InsertMailbox("INBOX", usr.Id)
		if e != nil {
			t.Error(e)
		}
		if mbx.Id == 0 {
			t.Errorf("Expected mailbox to have an ID set")
		}

		// READ
		mbxes, e := d.GetMailboxes(false, usr.Id)
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(mbxes, []*Mbx{mbx}); diff != "" {
			t.Errorf("Difference after fetching mailboxes with GetMailboxes %s", diff)
		}

		mbx2, e := d.GetMailboxByName("INBOX", usr.Id)
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(mbx2, mbx); diff != "" {
			t.Errorf("Difference after fetching mailboxes with GetMailboxByName %s", diff)
		}

		mbx3, e := d.GetMailboxById(mbx.Id)
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(mbx3, mbx); diff != "" {
			t.Errorf("Difference after fetching mailboxes with GetMailboxById %s", diff)
		}

		// UPDATE
		e = d.SetMailboxSubscribed(-1, false)
		if e == nil {
			t.Error("Expected error updating non-existent mailbox")
		}

		e = d.SetMailboxSubscribed(mbx.Id, false)
		if e != nil {
			t.Error(e)
		}

		mbxes2, e := d.GetMailboxes(true, usr.Id)
		if e != nil {
			t.Error(e)
		}
		if len(mbxes2) > 0 {
			t.Error("Expecting no subscribed mailboxes after update")
		}

		mbxes3, e := d.GetMailboxes(false, usr.Id)
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(mbxes3, []*Mbx{mbx}); diff != "" {
			t.Errorf("Difference looking for non-subscribed mailboxes with GetMailboxes %s", diff)
		}

		e = d.RenameMailbox(usr.Id, "FOO", "BAR")
		if e == nil {
			t.Error("Expected error renaming non-existent mailbox")
		}

		e = d.RenameMailbox(usr.Id, "INBOX", "SOMETHING")
		if e != nil {
			t.Error(e)
		}

		mbx.Name = "SOMETHING"
		mbx2, _ = d.GetMailboxById(mbx.Id)
		if diff := cmp.Diff(mbx, mbx2); diff != "" {
			t.Errorf("Difference after updating mailbox name")
		}

		// DELETE
		e = d.DeleteMailbox("BAR", usr.Id)
		if e == nil {
			t.Error("Expected error deleting non-existent mailbox")
		}

		e = d.DeleteMailbox("SOMETHING", usr.Id)
		if e != nil {
			t.Error(e)
		}

		_, e = d.GetMailboxById(mbx.Id)
		if e != NotFound {
			t.Error("Expected NotFound when looking for non-existent mailbox")
		}
	})
}

func TestQueueCrud(t *testing.T) {
	withDb(t, func(t *testing.T, d Database) {
		msgs, e := d.GetQueue()
		if e != nil {
			t.Error(e)
		}
		if len(msgs) > 0 {
			t.Error("Expected empty queue")
		}

		// CREATE
		msg, e := d.InsertQueue("foo@bar.com", "bar@foo.com", []byte("hello"), date)
		if e != nil {
			t.Error(e)
		}

		// READ
		msgs, e = d.GetQueue()
		if e != nil {
			t.Error(e)
		}
		if diff := cmp.Diff(msgs, []*QueuedMsg{msg}); diff != "" {
			t.Errorf("Difference in queued messages after retrieval %s", diff)
		}

		// UPDATE
		e = d.IncrementRetries(msg.Id)
		if e != nil {
			t.Error(e)
		}

		msgs, e = d.GetQueue()
		msg.Retries = 1
		if diff := cmp.Diff(msgs, []*QueuedMsg{msg}); diff != "" {
			t.Errorf("Difference in queued messages after updating retries %s", diff)
		}

		e = d.DeleteQueue(0)
		if e == nil {
			t.Error("Expected error deleting non-existent queue")
		}

		e = d.DeleteQueue(msg.Id)
		if e != nil {
			t.Error(e)
		}

		msgs, e = d.GetQueue()
		if len(msgs) > 0 {
			t.Error("Expected empty queue after deletion")
		}
	})
}
