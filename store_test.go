package main

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func BenchmarkPut(b *testing.B) {
	store, db, err := openTestStore(b)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := store.Put(context.Background(), "test", 0, 0, 2*time.Minute, "hello")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkReserve(b *testing.B) {
	store, db, err := openTestStore(b)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	for range 1000 {
		_, err = store.Put(context.Background(), "test", 0, 0, 2*time.Minute, "hello")
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := store.Reserve(context.Background(), "test")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDelete(b *testing.B) {
	store, db, err := openTestStore(b)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	for range 1000 {
		_, err = store.Put(context.Background(), "test", 0, 0, 2*time.Minute, "hello")
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := store.Delete(context.Background(), "test", 1)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func openTestStore(b *testing.B) (Store, *sql.DB, error) {
	tempdir := b.TempDir()
	db, err := sql.Open("sqlite3", tempdir+"/rebound.db?_mutex=no")
	if err != nil {
		return nil, nil, err
	}

	if _, err := db.Exec(initQuery); err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(1)
	return NewStore(db), db, nil
}
