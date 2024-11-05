package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return err
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

type QueueServer struct {
	http.Handler
	js         *JobStore
	defaultTTR time.Duration
}

func NewQueueServer(js *JobStore, ttr time.Duration) *QueueServer {
	var srv QueueServer

	srv.js = js
	srv.defaultTTR = ttr

	mux := http.NewServeMux()

	mux.HandleFunc("POST /{queue}", srv.handlePut)
	mux.HandleFunc("GET /{queue}", srv.handleReserve)
	mux.HandleFunc("DELETE /{queue}/{id}", srv.handleDelete)

	srv.Handler = mux
	return &srv
}

func (s *QueueServer) handlePut(w http.ResponseWriter, r *http.Request) {
	queue := r.PathValue("queue")
	if queue == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	type RequestBody struct {
		Priority int    `json:"priority"`
		Delay    string `json:"delay"`
		TTR      string `json:"ttr"`
		Body     string `json:"body"`
	}

	body, err := decode[RequestBody](r)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Default time.Duration is 0 seconds which is fine.
	delay, _ := time.ParseDuration(body.Delay)
	ttr, _ := time.ParseDuration(body.TTR)

	if ttr <= 0 {
		ttr = s.defaultTTR
	}

	id, err := s.js.Put(r.Context(), queue, body.Priority, delay, ttr, body.Body)
	if err != nil {
		http.Error(w, "Failed to insert message", http.StatusInternalServerError)
		return
	}

	responseBody := struct {
		Id int64 `json:"id"`
	}{Id: id}

	if err := encode(w, http.StatusCreated, responseBody); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *QueueServer) handleReserve(w http.ResponseWriter, r *http.Request) {
	queue := r.PathValue("queue")
	if queue == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	job, err := s.js.Reserve(r.Context(), queue)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Failed to reserve message", http.StatusInternalServerError)
		return
	}

	if err := encode(w, http.StatusOK, job); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *QueueServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	queue := r.PathValue("queue")
	if queue == "" {
		http.Error(w, "Queue name is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	if err := s.js.Delete(r.Context(), queue, id); err != nil {
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
