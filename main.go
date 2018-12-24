package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"log"
)

const SqlitePathKey = "DB_PATH"

func SetDefaults() {
	viper.SetDefault(SqlitePathKey, "henrymail.db")
}

func main() {
	SetDefaults()

	db, err := sql.Open("sqlite3", viper.GetString(SqlitePathKey))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
}
