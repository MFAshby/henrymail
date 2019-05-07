package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"henrymail/embedded"
	"log"
)

//go:generate ./generate_database.sh

var DB *sql.DB

func OpenDatabase() {
	var err error
	DB, err = sql.Open("sqlite3", "henrymail.db")
	if err != nil {
		log.Fatal(err)
	}
	initSqlBytes, err := embedded.GetEmbeddedContent().GetContents("/database/generate_schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	_, err = DB.Exec(string(initSqlBytes))
	if err != nil {
		log.Fatal(err)
	}
}

func Transact(db *sql.DB, txFunc func(*sql.Tx) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	err = txFunc(tx)
	return err
}
