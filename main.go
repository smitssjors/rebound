package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed init.sql
var initQuery string

func openDB(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./rebound.db?_mutex=no")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, initQuery); err != nil {
		return nil, err
	}

	return db, nil
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	logger := log.Default()

	db, err := openDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	store := NewStore(db)

	srv := NewServer(logger, store)
	httpServer := &http.Server{Addr: "0.0.0.0:3000", Handler: srv}

	// ListenAndServe blocks until the server is shut down.
	// Thus we run it in a goroutine and wait for the interrupt signal
	// on the main thread.
	go func() {
		log.Printf("listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("failed to listen and serve: %v", err)
		}
	}()

	// Wait for the interrupt signal.
	<-ctx.Done()

	// 10 second deadline for closing all connnections.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Print("shutting down")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %v", err)
	}

	return nil
}

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatalf("error: %v", err)
	}
}
