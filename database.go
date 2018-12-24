package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Database interface {
}

type sqliteDb struct {
	db *sql.DB
}

func NewDatabase() Database {
	db, err := sql.Open("sqlite3", GetString(SqlitePathKey))
	if err != nil {
		log.Fatal(err)
	}
	return &sqliteDb{db: db}
}
