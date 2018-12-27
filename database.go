package main

import (
	"database/sql"
	"errors"
	"github.com/emersion/go-imap"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	"time"
)

// Interface
type Database interface {
	InsertUser(email string, passwordBytes []byte) (*Usr, error)
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
}

var NotFound = errors.New("not found")

type sqliteDb struct {
	db *sql.DB
}

func (db *sqliteDb) SetMessageFlags(msgId int64, flags []string) error {
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

func (db *sqliteDb) DeleteMessage(msgId int64) error {
	_, e := db.db.Exec(`
		DELETE FROM messages
		WHERE id = ?
	`, msgId)
	return e
}

func (db *sqliteDb) RenameMailbox(userId int64, originalName, newName string) error {
	return checkOneRowAffected(db.db.Exec(`
		UPDATE mailboxes
		SET name = ?
		WHERE userid = ?
		AND name = ?
	`, newName, userId, originalName))
}

func (db *sqliteDb) SetMailboxSubscribed(mbxId int64, subscribed bool) error {
	return checkOneRowAffected(db.db.Exec(`
		UPDATE mailboxes
		SET subscribed = ?
		WHERE id = ?
	`, subscribed, mbxId))
}

func (db *sqliteDb) DeleteQueue(queueId int64) error {
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM queue 
		WHERE id = ?
		`, queueId))
}

func (db *sqliteDb) IncrementRetries(queueId int64) error {
	return checkOneRowAffected(db.db.Exec(`
		UPDATE queue
	  	SET retries = retries + 1
		WHERE id = ?
		`, queueId))
}

func (db *sqliteDb) GetQueue() ([]*QueuedMsg, error) {
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

func (db *sqliteDb) InsertQueue(from, to string, content []byte, timestamp time.Time) (*QueuedMsg, error) {
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

func (db *sqliteDb) GetInboxId(email string) (int64, error) {
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

func (db *sqliteDb) GetMessages(mbxId int64, lowerUid, upperUid int) ([]*Msg, error) {
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

func (db *sqliteDb) InsertMessage(content []byte, flags []string, mbxId int64, timestamp time.Time) (*Msg, error) {
	tx, e := db.db.Begin()
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()

	var uidNext uint32
	e = tx.QueryRow(`
		SELECT uidnext 
		FROM mailboxes 
		WHERE id = ?
	`, mbxId).Scan(&uidNext)
	if e != nil {
		return nil, e
	}

	msg := &Msg{
		MbxId:     mbxId,
		Uid:       uidNext,
		Content:   content,
		Flags:     flags,
		Timestamp: timestamp,
	}
	res, e := tx.Exec(`
		INSERT INTO messages (mailboxid, content, uid, ts) 
		VALUES (?, ?, ?, ?)
	`, mbxId, content, uidNext, timestamp)
	if checkErrorsSetId(&msg.HasId, res, e) != nil {
		return nil, e
	}

	for _, flag := range flags {
		_, e := tx.Exec(`
			INSERT INTO messageflags (messageid, flag) 
			VALUES (?, ?)
		`, msg.Id, flag)
		if e != nil {
			return nil, e
		}
	}

	_, e = tx.Exec(`
		UPDATE mailboxes 
		SET uidnext = ? 
		WHERE id = ?
	`, uidNext+1, mbxId)
	if e != nil {
		return nil, e
	}

	e = tx.Commit()
	if e != nil {
		return nil, e
	}
	return msg, nil
}

func (db *sqliteDb) InsertMailbox(name string, usrId int64) (*Mbx, error) {
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

func (db *sqliteDb) GetMailboxes(subscribed bool, usrId int64) ([]*Mbx, error) {
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

func (db *sqliteDb) GetMailboxById(id int64) (*Mbx, error) {
	row := db.db.QueryRow(`
		SELECT id, userid, name, uidnext, uidvalidity 
		FROM mailboxes 
		WHERE id = ?
	`, id)
	return db.readMailbox(row)
}

func (db *sqliteDb) GetMailboxByName(name string, usrId int64) (*Mbx, error) {
	row := db.db.QueryRow(`
		SELECT id, userid, name, uidnext, uidvalidity 
		FROM mailboxes 
		WHERE userid = ? 
		AND name = ?
	`, usrId, name)
	return db.readMailbox(row)
}

func (db *sqliteDb) readMailbox(row *sql.Row) (*Mbx, error) {
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

func (db *sqliteDb) DeleteMailbox(name string, usrId int64) error {
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM mailboxes 
		WHERE name = ? 
		AND userid = ?
	`, name, usrId))
}

func (db *sqliteDb) DeleteUser(email string) error {
	return checkOneRowAffected(db.db.Exec(`
		DELETE FROM users 
		WHERE email = ?
	`, email))
}

func (db *sqliteDb) GetUsers() ([]*Usr, error) {
	rows, e := db.db.Query(`
		SELECT id, email 
		FROM users
	`)
	if e != nil {
		return nil, e
	}
	users := make([]*Usr, 0)
	for rows.Next() {
		u := &Usr{}
		e := rows.Scan(&u.Id, &u.Email)
		if e != nil {
			return nil, e
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *sqliteDb) GetUserAndPassword(email string) (*Usr, []byte, error) {
	row := db.db.QueryRow(`
		SELECT id, email, passwordBytes 
		FROM users 
		WHERE Email = ?
	`, email)
	u := &Usr{}
	var pw []byte
	e := row.Scan(&u.Id, &u.Email, &pw)
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

func (db *sqliteDb) InsertUser(email string, passwordBytes []byte) (*Usr, error) {
	usr := &Usr{
		Email: email,
	}
	res, e := db.db.Exec(`
		INSERT INTO users (email, passwordBytes) 
		VALUES (?, ?)
	`, email, passwordBytes)
	return usr, checkErrorsSetId(&usr.HasId, res, e)
}

func NewDatabase() Database {
	db, err := sql.Open("sqlite3", GetString(SqlitePathKey))
	if err != nil {
		log.Fatal(err)
	}
	initSql := `
		PRAGMA foreign_keys = ON;
		--DROP TABLE IF EXISTS users;
		--DROP TABLE IF EXISTS mailboxes;
		--DROP TABLE IF EXISTS messages;
		--DROP TABLE IF EXISTS messageflags;
		--DROP TABLE IF EXISTS queue;

		CREATE TABLE IF NOT EXISTS users (
			id integer primary key,
			email text, 
			passwordBytes blob
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
	return &sqliteDb{db: db}
}
