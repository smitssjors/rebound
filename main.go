package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func run(ctx context.Context, getenv func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	db_file := getenv("DB_FILE")
	if db_file == "" {
		db_file = "./rebound.db"
	}
	jobStore, err := OpenJobStore(db_file)
	if err != nil {
		return fmt.Errorf("failed to open job store: %v", err)
	}
	defer jobStore.Close()

	ttr, err := time.ParseDuration(getenv("DEFAULT_TTR"))
	if err != nil {
		ttr = 2 * time.Minute
	}

	srv := NewQueueServer(jobStore, ttr)
	httpServer := &http.Server{Addr: ":3000", Handler: srv}

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

	log.Println("shutting down")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %v", err)
	}

	return nil
}

func main() {
	ctx := context.Background()

	if err := run(ctx, os.Getenv); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
