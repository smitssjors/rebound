package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func NewServer(logger *log.Logger, store Store) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /{queue}", handlePut(logger, store))
	mux.Handle("GET /{queue}", handleReserve(logger, store))
	mux.Handle("DELETE /{queue}/{id}", handleDelete(logger, store))

	return mux
}

func handlePut(l *log.Logger, s Store) http.Handler {
	type request struct {
		Priority int    `json:"priority"`
		Delay    string `json:"delay"`
		TTR      string `json:"ttr"`
		Body     string `json:"body"`
	}

	type response struct {
		Id Id `json:"id"`
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			queue := r.PathValue("queue")
			if queue == "" {
				l.Printf("invalid queue name: %s", queue)
				http.Error(w, "Queue name is required", http.StatusBadRequest)
				return
			}

			var req request
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				l.Printf("invalid JSON: %v", err)
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			delay, _ := time.ParseDuration(req.Delay)
			ttr, _ := time.ParseDuration(req.TTR)
			if ttr == 0 {
				ttr = 2 * time.Minute
			}

			id, err := s.Put(r.Context(), queue, req.Priority, delay, ttr, req.Body)
			if err != nil {
				l.Printf("failed to insert message: %v", err)
				http.Error(w, "Failed to insert message", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(response{Id: id}); err != nil {
				l.Printf("failed to encode response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		},
	)
}

func handleReserve(l *log.Logger, s Store) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			queue := r.PathValue("queue")
			if queue == "" {
				l.Printf("invalid queue name: %s", queue)
				http.Error(w, "Queue name is required", http.StatusBadRequest)
				return
			}

			job, err := s.Reserve(r.Context(), queue)
			if err != nil {
				l.Printf("failed to reserve message: %v", err)
			}
			if job == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if err := json.NewEncoder(w).Encode(job); err != nil {
				l.Printf("failed to encode response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		},
	)
}

func handleDelete(l *log.Logger, s Store) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			queue := r.PathValue("queue")
			if queue == "" {
				l.Printf("invalid queue name: %s", queue)
				http.Error(w, "Queue name is required", http.StatusBadRequest)
				return
			}

			id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
			if err != nil {
				l.Printf("invalid job ID: %v", err)
				http.Error(w, "Job ID is required", http.StatusBadRequest)
				return
			}

			if err := s.Delete(r.Context(), queue, Id(id)); err != nil {
				l.Printf("failed to delete message: %v", err)
				http.Error(w, "Failed to delete message", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		},
	)
}
