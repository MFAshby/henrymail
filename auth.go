package main

import (
	"golang.org/x/crypto/bcrypt"
)

type Login interface {
	Login(email, password string) (*Usr, error)
	NewUser(email, password string) (*Usr, error)
}

type dbLogin struct {
	db Database
}

func NewLogin(db Database) Login {
	return &dbLogin{db: db}
}

func (db dbLogin) Login(email, password string) (*Usr, error) {
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

func (db dbLogin) NewUser(email, password string) (*Usr, error) {
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return nil, e
	}
	usr, e := db.db.InsertUser(email, passwordBytes)
	if e != nil {
		return nil, e
	}
	return usr, nil
}
