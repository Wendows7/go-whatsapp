package database

import (
	"database/sql"
	"go-whatsapp/config"
)

var (
	env      = config.LoadEnv()
	hostname = env["DB_HOSTNAME"]
	port     = env["DB_PORT"]
	username = env["DB_USERNAME"]
	password = env["DB_PASSWORD"]
	database = env["DB_NAME"]
)

func DbUri() string {

	dbURI := "postgres://" + username + ":" + password + "@" + hostname + ":" + port + "/" + database + "?sslmode=disable"
	return dbURI
}

func InitDb() (*sql.DB, error) {
	dbURI := DbUri()
	db, err := sql.Open("postgres", dbURI)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
