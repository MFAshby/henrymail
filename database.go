package main

import (
	"database/sql"
	"errors"
	"github.com/emersion/go-imap"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	"sync"
	"time"
)

// Interface
type Database interface {
	InsertUser(email string, passwordBytes []byte, admin bool) (*Usr, error)
	GetUserAndPassword(email string) (*Usr, []byte, error)
	GetUsers() ([]*Usr, error)
	DeleteUser(email string) error

	InsertMailbox(name string, usrId int64) (*Mbx, error)
	GetMailboxes(subscribed bool, usrId int64) ([]*Mbx, error)
	GetMailboxByName(name string, usrId int64) (*Mbx, error)
	GetMailboxById(id int64) (*Mbx, error)
	GetInboxId(email string) (int64, error)
	SetMailboxSubscribed(mbxId int64, subscribed bool) error
	RenameMailbox(userId int64, originalName, newName string) error
	DeleteMailbox(name string, usrId int64) error

	InsertMessage(content []byte, flags []string, mbxId int64, timestamp time.Time) (*Msg, error)
	GetMessages(mbxId int64, lowerUid, upperUid int) ([]*Msg, error)
	SetMessageFlags(msgId int64, flags []string) error
	DeleteMessage(msgId int64) error

	InsertQueue(from, to string, content []byte, timestamp time.Time) (*QueuedMsg, error)
	GetQueue() ([]*QueuedMsg, error)
	IncrementRetries(queueId int64) error
	DeleteQueue(queueId int64) error
	SetUserPassword(email string, passwordBytes []byte) error
}

var NotFound = errors.New("not found")

type sqlDb struct {
	db  *sql.DB
	mut *sync.RWMutex
}

func (db *sqlDb) SetUserPassword(email string, passwordBytes []byte) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		UPDATE users
		SET passwordBytes = ?
		WHERE email = ?
	`, passwordBytes, email))
}

func (db *sqlDb) SetMessageFlags(msgId int64, flags []string) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	tx, e := db.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()

	_, e = db.db.Exec(`
		DELETE FROM messageflags
		WHERE messageid = ?
	`, msgId)
	if e != nil {
		return e
	}

	for _, flag := range flags {
		_, e = db.db.Exec(`
			INSERT INTO messageflags(messageid, flag)
			VALUES (?, ?)
		`, msgId, flag)
		if e != nil {
			return e
		}
	}

	return tx.Commit()
}

func (db *sqlDb) DeleteMessage(msgId int64) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	_, e := db.db.Exec(`
			DELETE FROM messageflags
			WHERE messageid = ?
		`, msgId)
	if e != nil {
		return e
	}
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM messages
		WHERE id = ?
	`, msgId))
}

func (db *sqlDb) RenameMailbox(userId int64, originalName, newName string) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		UPDATE mailboxes
		SET name = ?
		WHERE userid = ?
		AND name = ?
	`, newName, userId, originalName))
}

func (db *sqlDb) SetMailboxSubscribed(mbxId int64, subscribed bool) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		UPDATE mailboxes
		SET subscribed = ?
		WHERE id = ?
	`, subscribed, mbxId))
}

func (db *sqlDb) DeleteQueue(queueId int64) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM queue 
		WHERE id = ?
		`, queueId))
}

func (db *sqlDb) IncrementRetries(queueId int64) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		UPDATE queue
	  	SET retries = retries + 1
		WHERE id = ?
		`, queueId))
}

func (db *sqlDb) GetQueue() ([]*QueuedMsg, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	rows, e := db.db.Query(`
		SELECT id, msgfrom, msgto, ts, content, retries
		FROM queue
	`)
	if e != nil {
		return nil, e
	}
	var queue []*QueuedMsg
	for rows.Next() {
		msg := &QueuedMsg{}
		e := rows.Scan(&msg.Id, &msg.From, &msg.To, &msg.Timestamp, &msg.Content, &msg.Retries)
		if e != nil {
			return nil, e
		}
		queue = append(queue, msg)
	}
	return queue, nil
}

func checkErrorsSetId(o *HasId, r sql.Result, e error) error {
	if e != nil {
		return e
	}
	i, e := r.LastInsertId()
	if e != nil {
		return e
	}
	o.Id = i
	return nil
}

func (db *sqlDb) InsertQueue(from, to string, content []byte, timestamp time.Time) (*QueuedMsg, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	msg := &QueuedMsg{
		To:        to,
		From:      from,
		Retries:   0,
		Content:   content,
		Timestamp: timestamp,
	}
	r, e := db.db.Exec(`
		INSERT INTO queue(msgfrom, msgto, ts, content)
		VALUES (?, ?, ?, ?)
	`, from, to, timestamp, content)
	return msg, checkErrorsSetId(&msg.HasId, r, e)
}

func (db *sqlDb) GetInboxId(email string) (int64, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	var ibxId int64
	e := db.db.QueryRow(`
		SELECT mbx.id 
		FROM mailboxes mbx, users u 
		WHERE mbx.userid = u.id
		AND u.email = ? 
	`, email).Scan(&ibxId)
	if e == sql.ErrNoRows {
		e = NotFound
	}
	return ibxId, e
}

func (db *sqlDb) GetMessages(mbxId int64, lowerUid, upperUid int) ([]*Msg, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	var params []interface{}
	sb := strings.Builder{}
	sb.WriteString(`
		SELECT id, mailboxid, content, uid, ts 
		FROM messages 
		WHERE mailboxid = ?
	`)
	params = append(params, mbxId)
	if lowerUid > -1 {
		sb.WriteString(`
			AND uid >= ?
		`)
		params = append(params, lowerUid)
	}
	if upperUid > -1 {
		sb.WriteString(`
			AND uid <= ?
		`)
		params = append(params, upperUid)
	}
	rows, e := db.db.Query(sb.String(), params...)
	if e != nil {
		return nil, e
	}
	msgs := []*Msg{}
	for rows.Next() {
		msg := &Msg{}
		e := rows.Scan(&msg.Id, &msg.MbxId, &msg.Content, &msg.Uid, &msg.Timestamp)
		if e != nil {
			return nil, e
		}

		flagRows, e := db.db.Query(`
			SELECT flag 
			FROM messageflags 
			WHERE messageid = ?
		`, msg.Id)
		if e != nil {
			return nil, e
		}
		msg.Flags = make([]string, 0)
		for flagRows.Next() {
			var flag string
			e := flagRows.Scan(&flag)
			if e != nil {
				return nil, e
			}
			msg.Flags = append(msg.Flags, flag)
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func (db *sqlDb) InsertMessage(content []byte, flags []string, mbxId int64, timestamp time.Time) (*Msg, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	msg := &Msg{
		MbxId:     mbxId,
		Content:   content,
		Flags:     flags,
		Timestamp: timestamp,
	}
	e := transact(db.db, func(tx *sql.Tx) error {
		e := tx.QueryRow(`
			SELECT uidnext 
			FROM mailboxes 
			WHERE id = ?
		`, mbxId).Scan(&msg.Uid)
		if e != nil {
			return e
		}

		res, e := tx.Exec(`
			INSERT INTO messages (mailboxid, content, uid, ts) 
			VALUES (?, ?, ?, ?)
		`, mbxId, content, msg.Uid, timestamp)
		if checkErrorsSetId(&msg.HasId, res, e) != nil {
			return e
		}

		for _, flag := range flags {
			_, e := tx.Exec(`
				INSERT INTO messageflags (messageid, flag) 
				VALUES (?, ?)
			`, msg.Id, flag)
			if e != nil {
				return e
			}
		}

		_, e = tx.Exec(`
			UPDATE mailboxes 
			SET uidnext = ? 
			WHERE id = ?
		`, msg.Uid+1, mbxId)
		return e
	})
	return msg, e
}

func transact(db *sql.DB, txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
		}
	}()
	err = txFunc(tx)
	return err
}

func (db *sqlDb) InsertMailbox(name string, usrId int64) (*Mbx, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	mbx := &Mbx{
		Name:        name,
		UserId:      usrId,
		UidNext:     1,
		UidValidity: 1,
	}
	res, e := db.db.Exec(`
		INSERT INTO mailboxes (userid, name) 
		VALUES (?, ?)
	`, usrId, name)
	return mbx, checkErrorsSetId(&mbx.HasId, res, e)
}

func (db *sqlDb) GetMailboxes(subscribed bool, usrId int64) ([]*Mbx, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	sq := new(strings.Builder)
	var params []interface{}
	sq.WriteString(`
		SELECT id, userid, name, uidnext, uidvalidity 
		FROM mailboxes 
		WHERE userid = ?
	`)
	params = append(params, usrId)
	if subscribed {
		sq.WriteString(`
			AND subscribed = true			
		`)
	}
	rows, e := db.db.Query(sq.String(), params...)
	if e != nil {
		return nil, e
	}
	mbxs := make([]*Mbx, 0)
	for rows.Next() {
		mbx := &Mbx{}
		e := rows.Scan(&mbx.Id, &mbx.UserId, &mbx.Name, &mbx.UidNext, &mbx.UidValidity)
		if e != nil {
			return nil, e
		}
		mbxs = append(mbxs, mbx)
	}
	return mbxs, nil
}

func (db *sqlDb) GetMailboxById(id int64) (*Mbx, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	row := db.db.QueryRow(`
		SELECT id, userid, name, uidnext, uidvalidity 
		FROM mailboxes 
		WHERE id = ?
	`, id)
	return db.readMailbox(row)
}

func (db *sqlDb) GetMailboxByName(name string, usrId int64) (*Mbx, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	row := db.db.QueryRow(`
		SELECT id, userid, name, uidnext, uidvalidity 
		FROM mailboxes 
		WHERE userid = ? 
		AND name = ?
	`, usrId, name)
	return db.readMailbox(row)
}

func (db *sqlDb) readMailbox(row *sql.Row) (*Mbx, error) {
	mbx := &Mbx{}
	e := row.Scan(&mbx.Id, &mbx.UserId, &mbx.Name, &mbx.UidNext, &mbx.UidValidity)
	if e == sql.ErrNoRows {
		return nil, NotFound
	}
	if e != nil {
		return nil, e
	}

	// Fill in data from other tables..
	e = db.db.QueryRow(`
		SELECT COUNT(*) 
		FROM messages m
		WHERE m.mailboxid = ?
	`, mbx.Id).Scan(&mbx.Messages)
	if e != nil {
		return nil, e
	}

	e = db.db.QueryRow(`
		SELECT COUNT(DISTINCT m.id) 
		FROM messages m, messageflags mf
		WHERE m.mailboxid = ?
		AND mf.messageid = m.id
		AND mf.flag = ? 
	`, mbx.Id, imap.RecentFlag).Scan(&mbx.Recent)
	if e != nil {
		return nil, e
	}

	e = db.db.QueryRow(`
	SELECT COUNT(DISTINCT m.id) 
		FROM messages m
		WHERE m.mailboxid = ?
		AND NOT EXISTS (
			SELECT 1 FROM messageflags mf
		 	WHERE mf.flag = ?
		 	AND mf.messageid = m.id
		)
	`, mbx.Id, imap.SeenFlag).Scan(&mbx.Unseen)
	if e != nil {
		return nil, e
	}
	return mbx, nil
}

func checkOneRowAffected(r sql.Result, e error) error {
	if e != nil {
		return e
	}
	i, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if i != 1 {
		return NotFound
	}
	return nil
}

func (db *sqlDb) DeleteMailbox(name string, usrId int64) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM mailboxes 
		WHERE name = ? 
		AND userid = ?
	`, name, usrId))
}

func (db *sqlDb) DeleteUser(email string) error {
	db.mut.Lock()
	defer db.mut.Unlock()
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM users 
		WHERE email = ?
	`, email))
}

func (db *sqlDb) GetUsers() ([]*Usr, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	rows, e := db.db.Query(`
		SELECT id, email, admin 
		FROM users
	`)
	if e != nil {
		return nil, e
	}
	users := make([]*Usr, 0)
	for rows.Next() {
		u := &Usr{}
		e := rows.Scan(&u.Id, &u.Email, &u.Admin)
		if e != nil {
			return nil, e
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *sqlDb) GetUserAndPassword(email string) (*Usr, []byte, error) {
	db.mut.RLock()
	defer db.mut.RUnlock()
	row := db.db.QueryRow(`
		SELECT id, email, passwordBytes, admin 
		FROM users 
		WHERE Email = ?
	`, email)
	u := &Usr{}
	var pw []byte
	e := row.Scan(&u.Id, &u.Email, &pw, &u.Admin)
	if e == sql.ErrNoRows {
		return nil, nil, NotFound
	}
	if e == sql.ErrNoRows {
		return nil, nil, errors.New("user not found")
	}
	if e != nil {
		return nil, nil, e
	}
	return u, pw, nil
}

func (db *sqlDb) InsertUser(email string, passwordBytes []byte, admin bool) (*Usr, error) {
	db.mut.Lock()
	defer db.mut.Unlock()
	usr := &Usr{
		Email: email,
	}
	res, e := db.db.Exec(`
		INSERT INTO users (email, passwordBytes, admin) 
		VALUES (?, ?, ?)
	`, email, passwordBytes, admin)
	return usr, checkErrorsSetId(&usr.HasId, res, e)
}

func NewDatabase() Database {
	db, err := sql.Open(GetString(DbDriverNameKey), GetString(DbConnectionStringKey))
	if err != nil {
		log.Fatal(err)
	}
	initSql := `
		PRAGMA foreign_keys = ON;

		CREATE TABLE IF NOT EXISTS users (
			id integer primary key,
			email text, 
			passwordBytes blob,
			admin bool
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (
			email
		);
		CREATE TABLE IF NOT EXISTS mailboxes (
			id integer primary key,
			userid integer,
			name text,
			uidnext integer default 1,
			uidvalidity integer default 1,
			subscribed bool default true,
			FOREIGN KEY(userid) REFERENCES users(id)
		);
		CREATE TABLE IF NOT EXISTS messages (
			id integer primary key,
			mailboxid integer,
			content blob,
			uid integer,
			ts timestamp, 
			FOREIGN KEY (mailboxid) REFERENCES mailboxes(id)
		);
		CREATE TABLE IF NOT EXISTS messageflags (
			id integer primary key,
			messageid integer not null,
			flag text,
			FOREIGN KEY (messageid) REFERENCES messages(id)
		);
		CREATE TABLE IF NOT EXISTS queue (
		  	id integer primary key,
		  	msgfrom text,
		  	msgto text,
		  	ts timestamp,
		  	retries integer default 0,
		  	content blob                             
		);
	`
	_, err = db.Exec(initSql)
	if err != nil {
		log.Fatal(err)
	}
	return &sqlDb{
		db:  db,
		mut: new(sync.RWMutex),
	}
}
