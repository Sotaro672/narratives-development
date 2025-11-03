package database

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
    Client *sql.DB
}

// NewConnection initializes PostgreSQL connection.
func NewConnection(host, port, user, password, dbname string) (*DB, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open DB: %w", err)
    }

    // Connection pool tuning
    db.SetConnMaxLifetime(30 * time.Minute)
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("failed to ping DB: %w", err)
    }

    log.Println("[DB] Connected to PostgreSQL successfully")
    return &DB{Client: db}, nil
}

// Graceful shutdown
func (d *DB) Close() error {
    if d == nil || d.Client == nil {
        return nil
    }
    return d.Client.Close()
}
