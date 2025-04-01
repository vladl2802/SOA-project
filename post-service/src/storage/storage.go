package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(databaseUrl string) (*Storage, error) {
	conn, err := pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to the database: %w", err)
	}
	_, err = conn.Exec(context.Background(), postsTableSchema())
	if err != nil {
		return nil, fmt.Errorf("couldn't create table Posts in the database: %w", err)
	}

	return &Storage{pool: conn}, nil
}

func (s *Storage) Begin(ctx context.Context) (Tx, error) {
	tx, err := s.pool.Begin(ctx)
	return Tx{tx: tx}, err
}

func (s *Storage) Close() {
	s.pool.Close()
}

type Tx struct {
	tx pgx.Tx
}

func (tx *Tx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx *Tx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}
