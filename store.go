package main

import (
	"context"
	"database/sql"
	"time"
)

type Id int64

type Job struct {
	Id       int64  `json:"id"`
	Queue    string `json:"queue"`
	Priority int    `json:"priority"`
	Body     string `json:"body"`
}

type Store interface {
	Put(ctx context.Context, queue string, priority int, delay, ttr time.Duration, body string) (Id, error)
	Reserve(ctx context.Context, queue string) (*Job, error)
	Delete(ctx context.Context, queue string, id Id) error
}

type sqliteStore struct {
	db *sql.DB
}

func (s *sqliteStore) Put(ctx context.Context, q string, pri int, delay time.Duration, ttr time.Duration, body string) (Id, error) {
	var id int64
	err := tx(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		res, err := tx.ExecContext(
			ctx,
			"INSERT INTO messages (queue, priority, locked_until, ttr, body) VALUES (?, ?, ?, ?, ?)",
			q,
			pri,
			time.Now().Add(delay).Unix(),
			int64(ttr.Round(time.Second).Seconds()),
			body,
		)

		if err != nil {
			return err
		}

		id, err = res.LastInsertId()
		return err
	})

	if err != nil {
		return 0, err
	}

	return Id(id), nil
}

func (s *sqliteStore) Reserve(ctx context.Context, q string) (*Job, error) {
	var job Job

	err := tx(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		now := time.Now().Unix()
		row := tx.QueryRowContext(
			ctx,
			"UPDATE messages SET locked_until = (? + ttr) WHERE id = (SELECT id FROM messages WHERE queue = ? AND locked_until <= ? ORDER BY priority DESC LIMIT 1) RETURNING id, queue, priority, body",
			now,
			q,
			now,
		)

		return row.Scan(&job.Id, &job.Queue, &job.Priority, &job.Body)

	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &job, nil
}

func (s *sqliteStore) Delete(ctx context.Context, q string, id Id) error {
	err := tx(ctx, s.db, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			"DELETE FROM messages WHERE queue = ? AND id = ?",
			q,
			id)

		return err
	})

	return err
}

func NewStore(db *sql.DB) Store {
	return &sqliteStore{db: db}
}

func tx(ctx context.Context, db *sql.DB, f func(context.Context, *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := f(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
