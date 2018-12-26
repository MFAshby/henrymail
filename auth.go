package main

import (
	"golang.org/x/crypto/bcrypt"
)

type Login interface {
	Login(email, password string) (*User, error)
	NewUser(email, password string) (*User, error)
}

type User struct {
	Email string
}

type dbLogin struct {
	db Database
}

func NewLogin(db Database) Login {
	login := &dbLogin{db: db}
	_, _ = login.NewUser("martin@test.com", "12345")
	_, _ = login.NewUser("someone@test.com", "12345")
	return login
}

func (db dbLogin) Login(email, password string) (*User, error) {
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

func (db dbLogin) NewUser(email, password string) (*User, error) {
	passwordBytes, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return nil, e
	}
	e = db.db.InsertUser(email, passwordBytes)
	if e != nil {
		return nil, e
	}
	return &User{Email: email}, nil
}
