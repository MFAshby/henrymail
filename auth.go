package main

import (
	"errors"
)

type Login interface {
	Login(username string, password string) (*User, error)
}

type User struct {
	username string
}

// Implementation, should be in a different file I guess
func NewLogin(db Database) Login {
	return &dbLogin{}
}

type dbLogin struct{}

func (dbLogin) Login(username, password string) (*User, error) {
	// TODO fetch the details off the database!
	if username != "username" || password != "password" {
		return nil, errors.New("Invalid username or password")
	}
	return &User{username: username}, nil
}
