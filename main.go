package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
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

func main() {
	dsn := fmt.Sprintf("file:%s?_txlock=IMMEDIATE&_mutex=NO", "./rebound.db")
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Panicf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(initQuery)
	if err != nil {
		log.Panicf("Failed to initialize database: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /{queue}", func(w http.ResponseWriter, r *http.Request) {
		queue := r.PathValue("queue")
		if queue == "" {
			http.Error(w, "Queue name is required", http.StatusBadRequest)
			return
		}

		var body struct {
			Priority int    `json:"priority"`
			Delay    string `json:"delay"`
			TTR      string `json:"ttr"`
			Body     string `json:"body"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		var delay time.Duration
		if body.Delay != "" {
			delay, err = time.ParseDuration(body.Delay)
			if err != nil {
				http.Error(w, "Invalid delay", http.StatusBadRequest)
				return
			}
		}

		var ttr time.Duration
		if body.TTR != "" {
			ttr, err = time.ParseDuration(body.TTR)
			if err != nil {
				http.Error(w, "Invalid time to run", http.StatusBadRequest)
				return
			}
		}
		if ttr <= 0 {
			ttr = 2 * time.Minute // Default TTR
		}

		res, err := db.ExecContext(
			r.Context(),
			"INSERT INTO messages (queue, priority, locked_until, ttr, body) VALUES (?, ?, ?, ?, ?)",
			queue,
			body.Priority,
			time.Now().Add(delay).Unix(),
			ttr.Round(time.Second).Seconds(),
			body.Body,
		)
		if err != nil {
			http.Error(w, "Failed to insert message", http.StatusInternalServerError)
			return
		}

		id, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to get last insert id", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": %d}`, id)
	})

	mux.HandleFunc("GET /{queue}", func(w http.ResponseWriter, r *http.Request) {
		queue := r.PathValue("queue")
		if queue == "" {
			http.Error(w, "Queue name is required", http.StatusBadRequest)
			return
		}

		now := time.Now().Unix()
		row := db.QueryRowContext(
			r.Context(),
			"UPDATE messages SET locked_until = (? + ttr) WHERE id = (SELECT id FROM messages WHERE queue = ? AND locked_until <= ? ORDER BY priority DESC LIMIT 1) RETURNING id, body",
			now,
			queue,
			now,
		)
		var message struct {
			Id   int64  `json:"id"`
			Body string `json:"body"`
		}

		err := row.Scan(&message.Id, &message.Body)
		if err == sql.ErrNoRows {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(message)
	})

	mux.HandleFunc("DELETE /{queue}/{id}", func(w http.ResponseWriter, r *http.Request) {
		queue := r.PathValue("queue")
		if queue == "" {
			http.Error(w, "Queue name is required", http.StatusBadRequest)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "Message ID is required", http.StatusBadRequest)
			return
		}

		_, err := db.ExecContext(
			r.Context(),
			"DELETE FROM messages WHERE queue = ? AND id = ?",
			queue,
			id,
		)
		if err != nil {
			http.Error(w, "Failed to delete message", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	srv := http.Server{Addr: ":3000", Handler: mux}

	// Handle graceful shutdown
	go func() {
		sigin := make(chan os.Signal, 1)
		signal.Notify(sigin, os.Interrupt)
		<-sigin

		log.Println("Shutting down...")

		if err := srv.Shutdown(context.Background()); err != nil {
			log.Panicf("Failed to shutdown server: %v", err)
		}

	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Panicf("Failed to start server: %v", err)
	}

	log.Println("Server shutdown, bye!")
}
