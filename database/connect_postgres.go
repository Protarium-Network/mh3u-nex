package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"

	"github.com/Protarium-Network/mh3u-nex/globals"
)

var Postgres *sql.DB

func ConnectPostgres() {
	var err error

	Postgres, err = sql.Open("postgres", os.Getenv("PN_MH3U_POSTGRES_URI"))
	if err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	if err = Postgres.Ping(); err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	globals.Logger.Success("Connected to Postgres!")

	initPostgres()
}
