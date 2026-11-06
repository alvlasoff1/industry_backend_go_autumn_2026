package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: map[string]Task{}}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	t := Task{
		ID:        strconv.FormatUint(r.seq, 10),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}
	r.tasks[t.ID] = t
	return t, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	return t, ok
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	arr := []Task{}
	for _, t := range r.tasks {
		if t.Done == done {
			arr = append(arr, t)
		}
	}
	r.mu.RUnlock()
	sort.Slice(arr, func(i, j int) bool {
		if !arr[i].UpdatedAt.Equal(arr[j].UpdatedAt) {
			return arr[i].UpdatedAt.After(arr[j].UpdatedAt)
		}
		return arr[i].ID < arr[j].ID
	})
	return arr
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if t.Done == done {
		return t, nil
	}
	t.Done = done
	t.UpdatedAt = r.clock.Now()
	r.tasks[id] = t
	return t, nil
}

type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if !strings.HasPrefix(p, "/tasks/") {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimPrefix(p, "/tasks/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPatch:
		h.handlePatch(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title *string `json:"title"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Title == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.repo.Create(*req.Title)
	if errors.Is(err, ErrInvalidTitle) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.repo.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	done := false
	vals, ok := r.URL.Query()["done"]
	if ok {
		if len(vals) != 1 {
			http.Error(w, "bad done", http.StatusBadRequest)
			return
		}
		switch vals[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			http.Error(w, "bad done", http.StatusBadRequest)
			return
		}
	}
	arr := h.repo.List(done)
	if arr == nil {
		arr = []Task{}
	}
	writeJSON(w, http.StatusOK, arr)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Done *bool `json:"done"`
	}
	if err := decodeStrictJSON(r.Body, &req); err != nil || req.Done == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.repo.SetDone(id, *req.Done)
	if errors.Is(err, ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var x any
	if err := dec.Decode(&x); err != io.EOF {
		return errors.New("extra data after json")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
