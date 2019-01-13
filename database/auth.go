package database

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"henrymail/config"
	"henrymail/model"
)

/**
 * A wrapper around the database for correctly handling login
 */
type Login interface {
	Login(email, password string) (*model.Usr, error)
	NewUser(email, password string, admin bool) (*model.Usr, error)
	ChangePassword(email, password, password2 string) error
}

type dbLogin struct {
	db Database
}

func NewLogin(db Database) Login {
	return &dbLogin{db: db}
}

func (db dbLogin) Login(email, password string) (*model.Usr, error) {
	user, passwordBytes, e := db.db.GetUserAndPassword(email)
	if e != nil {
		return nil, e
	}
	e = bcrypt.CompareHashAndPassword(passwordBytes, []byte(password))
	if e != nil {
		return nil, e
	}
	return user, nil
}

func (db dbLogin) NewUser(email, password string, admin bool) (*model.Usr, error) {
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return nil, e
	}
	usr, e := db.db.InsertUser(email, passwordBytes, admin)
	if e != nil {
		return nil, e
	}
	defaultMailboxes := config.GetStringSlice(config.DefaultMailboxesKey)
	for _, name := range defaultMailboxes {
		_, e := db.db.InsertMailbox(name, usr.Id)
		if e != nil {
			return nil, e
		}
	}
	return usr, nil
}

func (db dbLogin) ChangePassword(email, password, password2 string) error {
	// TODO password policy
	if password == "" {
		return errors.New("You must enter a password")
	}
	if password != password2 {
		return errors.New("Passwords don't match")
	}
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	return db.db.SetUserPassword(email, passwordBytes)
}
