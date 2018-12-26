package main

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Database interface {
	GetUserAndPassword(email string) (*User, []byte, error)
	InsertUser(email string, passwordBytes []byte) error
	GetUsers() ([]*User, error)
	DeleteUser(email string) error
}

type sqliteDb struct {
	db *sql.DB
}

func (db *sqliteDb) DeleteUser(email string) error {
	_, e := db.db.Exec(`DELETE FROM users WHERE email = ?`, email)
	return e
}

func (db *sqliteDb) GetUsers() ([]*User, error) {
	rows, e := db.db.Query("SELECT email FROM users")
	if e != nil {
		return nil, e
	}
	users := make([]*User, 0)
	for rows.Next() {
		u := &User{}
		e := rows.Scan(&u.Email)
		if e != nil {
			return nil, e
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *sqliteDb) GetUserAndPassword(email string) (*User, []byte, error) {
	row := db.db.QueryRow(`SELECT Email, passwordBytes FROM users WHERE Email = ?`, email)
	u := &User{}
	var pw []byte
	e := row.Scan(&u.Email, &pw)
	if e == sql.ErrNoRows {
		return nil, nil, errors.New("user not found")
	}
	if e != nil {
		return nil, nil, e
	}
	return u, pw, nil
}

func (db *sqliteDb) InsertUser(email string, passwordBytes []byte) error {
	_, e := db.db.Exec(`INSERT INTO users VALUES (?, ?)`, email, passwordBytes)
	return e
}

func NewDatabase() Database {
	db, err := sql.Open("sqlite3", GetString(SqlitePathKey))
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (Email text, passwordBytes blob)
	`)
	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS users_email ON users (email)
	`)
	if err != nil {
		log.Fatal(err)
	}
	return &sqliteDb{db: db}
}
