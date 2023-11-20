//go:build integration
// +build integration

package data

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
)

var testModels Models

func TestMain(m *testing.M) {
	var cfg DBCfg
	ParseDBCfg(&cfg)
	flag.Parse()

	testDB, err := OpenDB(cfg)
	fmt.Printf("cfg test db: %+v\n", cfg)

	if err != nil {
		log.Fatalf("make sure to set up env vars to run the integration tests. err: %s", err)
	}

	testModels = NewModels(testDB)
	os.Exit(m.Run())
}
