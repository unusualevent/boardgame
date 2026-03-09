package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	MaxConns     int           `json:"max_conns"`
}

type Server struct {
	config  Config
	mux     *http.ServeMux
	clients map[string]*Client
	mu      sync.RWMutex
	done    chan struct{}
}

type Client struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Connected time.Time `json:"connected"`
	LastPing  time.Time `json:"last_ping"`
}

type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewServer(cfg Config) *Server {
	s := &Server{
		config:  cfg,
		mux:     http.NewServeMux(),
		clients: make(map[string]*Client),
		done:    make(chan struct{}),
	}
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/clients", s.handleClients)
	s.mux.HandleFunc("/api/clients/", s.handleClientByID)
	s.mux.HandleFunc("/api/broadcast", s.handleBroadcast)
	return s
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	go func() {
		select {
		case <-ctx.Done():
			srv.Shutdown(context.Background())
		case <-s.done:
			srv.Shutdown(context.Background())
		}
	}()

	log.Printf("server starting on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (s *Server) Stop() {
	close(s.done)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSON(w, http.StatusMethodNotAllowed, Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "method not allowed",
		})
		return
	}
	s.mu.RLock()
	count := len(s.clients)
	s.mu.RUnlock()

	s.writeJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "ok",
		Data: map[string]interface{}{
			"uptime":  time.Since(time.Now()).String(),
			"clients": count,
		},
	})
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		clients := make([]*Client, 0, len(s.clients))
		for _, c := range s.clients {
			clients = append(clients, c)
		}
		s.mu.RUnlock()

		s.writeJSON(w, http.StatusOK, Response{
			Status:  http.StatusOK,
			Message: "ok",
			Data:    clients,
		})

	case http.MethodPost:
		var c Client
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			s.writeJSON(w, http.StatusBadRequest, Response{
				Status:  http.StatusBadRequest,
				Message: "invalid request body",
				Error:   err.Error(),
			})
			return
		}
		c.Connected = time.Now()
		c.LastPing = time.Now()

		s.mu.Lock()
		if len(s.clients) >= s.config.MaxConns {
			s.mu.Unlock()
			s.writeJSON(w, http.StatusServiceUnavailable, Response{
				Status:  http.StatusServiceUnavailable,
				Message: "max connections reached",
			})
			return
		}
		s.clients[c.ID] = &c
		s.mu.Unlock()

		s.writeJSON(w, http.StatusCreated, Response{
			Status:  http.StatusCreated,
			Message: "client registered",
			Data:    c,
		})

	default:
		s.writeJSON(w, http.StatusMethodNotAllowed, Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "method not allowed",
		})
	}
}

func (s *Server) handleClientByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/clients/"):]
	if id == "" {
		s.writeJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "client id required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		c, ok := s.clients[id]
		s.mu.RUnlock()
		if !ok {
			s.writeJSON(w, http.StatusNotFound, Response{
				Status:  http.StatusNotFound,
				Message: "client not found",
			})
			return
		}
		s.writeJSON(w, http.StatusOK, Response{
			Status:  http.StatusOK,
			Message: "ok",
			Data:    c,
		})

	case http.MethodDelete:
		s.mu.Lock()
		if _, ok := s.clients[id]; !ok {
			s.mu.Unlock()
			s.writeJSON(w, http.StatusNotFound, Response{
				Status:  http.StatusNotFound,
				Message: "client not found",
			})
			return
		}
		delete(s.clients, id)
		s.mu.Unlock()

		s.writeJSON(w, http.StatusOK, Response{
			Status:  http.StatusOK,
			Message: "client removed",
		})

	default:
		s.writeJSON(w, http.StatusMethodNotAllowed, Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "method not allowed",
		})
	}
}

func (s *Server) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "method not allowed",
		})
		return
	}

	var msg struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		s.writeJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	s.mu.RLock()
	count := len(s.clients)
	s.mu.RUnlock()

	log.Printf("broadcast to %d clients: %s", count, msg.Text)

	s.writeJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: fmt.Sprintf("sent to %d clients", count),
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
