package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn         string
		maxConns    int
		maxIdleTime string
	}
}

type application struct {
	config config
	logger *log.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://greenlight:pa55word@localhost/greenlight?sslmode=disable", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	logger.Printf("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = int32(cfg.db.maxConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConnIdleTime = duration

	conn, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = conn.Ping(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}
