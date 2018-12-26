package main

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

// Need to control concurrency for write operations.
type Database interface {
	InsertUser(email string, passwordBytes []byte) (*Usr, error)
	GetUserAndPassword(email string) (*Usr, []byte, error)
	GetUsers() ([]*Usr, error)
	DeleteUser(email string) error

	InsertMailbox(name string, usrId int64) (*Mbx, error)
	GetMailboxes(subscribed bool, usrId int64) ([]*Mbx, error)
	GetMailbox(name string, usrId int64) (*Mbx, error)
	DeleteMailbox(name string, usrId int64) error

	InsertMessage(content []byte, flags []string, mbxId int64) (*Msg, error)
	GetMessages(mbxId int64, lowerUid uint32) ([]*Msg, error)
	GetInboxId(email string) (int64, error)
	InsertQueue(from, to string, content []byte, timestamp time.Time) error
	GetQueuedMsgs() ([]*QueuedMsg, error)
}

type sqliteDb struct {
	db *sql.DB
}

func (db *sqliteDb) GetQueuedMsgs() ([]*QueuedMsg, error) {
	panic("implement me")
}

func (db *sqliteDb) InsertQueue(from, to string, content []byte, timestamp time.Time) error {
	_, e := db.db.Exec(`
			INSERT INTO queue(msgfrom, msgto, timestamp, content)
			VALUES (?, ?, ?, ?)
		`, from, to, timestamp, content)
	return e
}

var NotFound = errors.New("not found")

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

func (db *sqliteDb) GetMessages(mbxId int64, lowerUid uint32) ([]*Msg, error) {
	rows, e := db.db.Query(`
			SELECT id, mailboxid, content, uid FROM messages WHERE mailboxid = ? AND uid >= ? 
		`, mbxId, lowerUid)
	if e != nil {
		return nil, e
	}
	msgs := make([]*Msg, 0)
	for rows.Next() {
		msg := &Msg{}
		e := rows.Scan(&msg.Id, &msg.MbxId, &msg.Content, &msg.Uid)
		if e != nil {
			return nil, e
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (db *sqliteDb) InsertMessage(content []byte, flags []string, mbxId int64) (*Msg, error) {
	tx, e := db.db.Begin()
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()

	var uidNext uint32
	e = tx.QueryRow(`SELECT uidnext FROM mailboxes WHERE id = ?`, mbxId).Scan(&uidNext)
	if e != nil {
		return nil, e
	}

	res, e := tx.Exec(`INSERT INTO messages (mailboxid, content, uid) VALUES (?, ?, ?)`, mbxId, content, uidNext)
	if e != nil {
		return nil, e
	}

	iid, e := res.LastInsertId()
	if e != nil {
		return nil, e
	}

	for flag := range flags {
		_, e := tx.Exec(`INSERT INTO messageflags (messageid, flag) VALUES (?, ?)`, iid, flag)
		if e != nil {
			return nil, e
		}
	}

	_, e = tx.Exec(`UPDATE mailboxes SET uidnext = ? WHERE id = ?`, uidNext+1, mbxId)
	if e != nil {
		return nil, e
	}

	e = tx.Commit()
	if e != nil {
		return nil, e
	}
	return &Msg{
		Id:      iid,
		MbxId:   mbxId,
		Content: content,
		Flags:   flags,
	}, nil
}

func (db *sqliteDb) InsertMailbox(name string, usrId int64) (*Mbx, error) {
	res, e := db.db.Exec(`INSERT INTO mailboxes (userid, name) VALUES (?, ?)`, usrId, name)
	if e != nil {
		return nil, e
	}
	iid, e := res.LastInsertId()
	if e != nil {
		return nil, e
	}
	return &Mbx{
		Id:     iid,
		Name:   name,
		UserId: usrId,
	}, nil
}

func (db *sqliteDb) GetMailboxes(subscribed bool, usrId int64) ([]*Mbx, error) {
	rows, e := db.db.Query(`
	SELECT id, userid, name, uidnext, uidvalidity 
		from mailboxes where userid = ?
	`, usrId)
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

func (db *sqliteDb) GetMailbox(name string, usrId int64) (*Mbx, error) {
	row := db.db.QueryRow(`SELECT id, userid, name, uidnext, uidvalidity from mailboxes where userid = ? and name = ?`, usrId, name)
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
		AND mf.flag = '\Recent'
	`, mbx.Id).Scan(&mbx.Recent)
	if e != nil {
		return nil, e
	}

	e = db.db.QueryRow(`
	SELECT COUNT(DISTINCT m.id) 
		FROM messages m
		WHERE m.mailboxid = ?
		AND NOT EXISTS (
			SELECT 1 FROM messageflags mf
		 	WHERE mf.flag = '\Seen'
		 	AND mf.messageid = m.id
		)
	`, mbx.Id).Scan(&mbx.Unseen)
	if e != nil {
		return nil, e
	}
	return mbx, nil
}

func (db *sqliteDb) DeleteMailbox(name string, usrId int64) error {
	_, e := db.db.Exec(`DELETE FROM mailboxes where name = ? and userid = ?`, name, usrId)
	return e
}

func (db *sqliteDb) DeleteUser(email string) error {
	_, e := db.db.Exec(`DELETE FROM users WHERE email = ?`, email)
	return e
}

func (db *sqliteDb) GetUsers() ([]*Usr, error) {
	rows, e := db.db.Query("SELECT id, email FROM users")
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
	row := db.db.QueryRow(`SELECT id, email, passwordBytes FROM users WHERE Email = ?`, email)
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
	res, e := db.db.Exec(`INSERT INTO users (email, passwordBytes) VALUES (?, ?)`,
		email, passwordBytes)
	if e != nil {
		return nil, e
	}
	iid, e := res.LastInsertId()
	if e != nil {
		return nil, e
	}
	return &Usr{
		Id:    iid,
		Email: email,
	}, nil
}

func NewDatabase() Database {
	db, err := sql.Open("sqlite3", GetString(SqlitePathKey))
	if err != nil {
		log.Fatal(err)
	}
	initSql := `
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS mailboxes;
		DROP TABLE IF EXISTS messages;
		DROP TABLE IF EXISTS messageflags;
		DROP TABLE IF EXISTS queue;
			
		CREATE TABLE IF NOT EXISTS users (
			id integer primary key,
			email text, 
			passwordBytes blob
		);
		CREATE UNIQUE INDEX IF NOT EXISTS users_email ON users (
			email
		);
		CREATE TABLE IF NOT EXISTS mailboxes (
			id integer primary key,
			userid integer,
			name text,
			uidnext integer default 1,
			uidvalidity integer default 1,
			FOREIGN KEY(userid) REFERENCES users(id)
		);
		CREATE TABLE IF NOT EXISTS messages (
			id integer primary key,
			mailboxid integer,
			content blob,
			uid integer, 
			FOREIGN KEY (mailboxid) REFERENCES mailboxes(id)
		);
		CREATE TABLE IF NOT EXISTS messageflags (
			id integer primary key,
			messageid integer,
			flag text,
			FOREIGN KEY (messageid) REFERENCES messages(id)
		);
		CREATE TABLE IF NOT EXISTS queue (
		  	id integer primary key,
		  	msgfrom text,
		  	msgto text,
		  	timestamp integer,
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
