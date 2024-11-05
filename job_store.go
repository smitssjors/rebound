package main

import (
	"context"
	"database/sql"
	_ "embed"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed init.sql
var initQuery string

type JobStore struct {
	db *sql.DB
}

func OpenJobStore(file string) (*JobStore, error) {
	// dsn := fmt.Sprintf("file:%s?_txlock=IMMEDIATE&_mutex=NO", "./rebound.db")
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(initQuery); err != nil {
		return nil, err
	}

	return &JobStore{db: db}, nil
}

func (j *JobStore) Close() error {
	return j.db.Close()
}

func (j *JobStore) Put(ctx context.Context, q string, pri int, delay, ttr time.Duration, body string) (int64, error) {
	res, err := j.db.ExecContext(
		ctx,
		"INSERT INTO messages (queue, priority, locked_until, ttr, body) VALUES (?, ?, ?, ?, ?)",
		q,
		pri,
		time.Now().Add(delay).Unix(),
		int64(ttr.Round(time.Second).Seconds()),
		body,
	)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

type Job struct {
	Id       int64  `json:"id"`
	Queue    string `json:"queue"`
	Priority int    `json:"priority"`
	Body     string `json:"body"`
}

func (j *JobStore) Reserve(ctx context.Context, q string) (*Job, error) {
	now := time.Now().Unix()
	row := j.db.QueryRowContext(
		ctx,
		"UPDATE messages SET locked_until = (? + ttr) WHERE id = (SELECT id FROM messages WHERE queue = ? AND locked_until <= ? ORDER BY priority DESC LIMIT 1) RETURNING id, queue, priority, body",
		now,
		q,
		now,
	)

	var job Job
	err := row.Scan(&job.Id, &job.Queue, &job.Priority, &job.Body)
	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (j *JobStore) Delete(ctx context.Context, q string, id int64) error {
	_, err := j.db.ExecContext(
		ctx,
		"DELETE FROM messages WHERE queue = ? AND id = ?",
		q,
		id)

	return err
}
