package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"henrymail/config"
	"henrymail/embedded"
	"log"
)

//go:generate ./generate_database.sh

func OpenDatabase() *sql.DB {
	db, err := sql.Open(config.GetString(config.DbDriverName), config.GetString(config.DbConnectionString))
	if err != nil {
		log.Fatal(err)
	}
	initSqlBytes, err := embedded.GetEmbeddedContent().GetContents("/database/generate_schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(string(initSqlBytes))
	if err != nil {
		log.Fatal(err)
	}
	return db
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
