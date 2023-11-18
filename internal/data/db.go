package data

import (
	"context"
	"database/sql"
	"flag"
	"time"
)

type DBCfg struct {
	Dsn           string
	MaxOpenConns  int
	MaxIdlesConns int
	MaxIdleTime   string
}

func OpenDB(cfg DBCfg) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdlesConns)

	duration, err := time.ParseDuration(cfg.MaxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func ParseDBCfg(cfg *DBCfg) {
	flag.StringVar(&cfg.Dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.MaxIdlesConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
}
