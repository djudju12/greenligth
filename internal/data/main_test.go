package data

import (
	"database/sql"
	"os"
	"testing"
)

var testDB struct {
	cfg DBCfg
	DB  *sql.DB
}

func TestMain(m *testing.M) {
	ParseDBCfg(&testDB.cfg)

	var err error
	testDB.DB, err = OpenDB(testDB.cfg)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
