package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Todo struct {
	ID      int    `json:"id"`
	Subject string `json:"subject"`
	Details string `json:"details"`
}

type todoInput struct {
	Subject string `json:"subject"`
	Details string `json:"details"`
}

type Controller struct {
	DB *pgxpool.Pool
}

func ListTodos(ctx context.Context, db *pgxpool.Pool) ([]Todo, error) {
	rows, err := db.Query(ctx, `SELECT id, subject, details FROM todos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Todo])
}

func FindTodo(ctx context.Context, db *pgxpool.Pool, id int) (Todo, error) {
	return oneTodo(ctx, db, `SELECT id, subject, details FROM todos WHERE id = $1`, id)
}

func InsertTodo(ctx context.Context, db *pgxpool.Pool, in todoInput) (Todo, error) {
	return oneTodo(ctx, db,
		`INSERT INTO todos (subject, details) VALUES ($1, $2) RETURNING id, subject, details`,
		in.Subject, in.Details)
}

func UpdateTodo(ctx context.Context, db *pgxpool.Pool, id int, in todoInput) (Todo, error) {
	return oneTodo(ctx, db,
		`UPDATE todos SET subject = $1, details = $2 WHERE id = $3 RETURNING id, subject, details`,
		in.Subject, in.Details, id)
}

// DeleteTodo reports pgx.ErrNoRows when no row had that id.
func DeleteTodo(ctx context.Context, db *pgxpool.Pool, id int) error {
	tag, err := db.Exec(ctx, `DELETE FROM todos WHERE id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

// oneTodo runs a query that yields one row, or pgx.ErrNoRows when nothing matched.
func oneTodo(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (Todo, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return Todo{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[Todo])
}

// idParam reads {id} from the URL. A non-numeric id becomes 0, which no row has, so it ends as 404.
func idParam(r *http.Request) int {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	return id
}

// readInput decodes and trims the body; it answers 400 itself when the body is unusable.
func readInput(w http.ResponseWriter, r *http.Request) (todoInput, bool) {
	var in todoInput
	if err := decode(r, &in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return in, false
	}
	in.Subject, in.Details = strings.TrimSpace(in.Subject), strings.TrimSpace(in.Details)
	if in.Subject == "" || in.Details == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "subject and details are required"})
		return in, false
	}
	return in, true
}

// reply sends value with the given status, or turns the error into 404 / 500 JSON.
func reply(w http.ResponseWriter, status int, value any, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
	case status == http.StatusNoContent:
		w.WriteHeader(status)
	default:
		writeJSON(w, status, value)
	}
}

func (c Controller) Index(w http.ResponseWriter, r *http.Request) {
	todos, err := ListTodos(r.Context(), c.DB)
	reply(w, http.StatusOK, todos, err)
}

func (c Controller) Show(w http.ResponseWriter, r *http.Request) {
	t, err := FindTodo(r.Context(), c.DB, idParam(r))
	reply(w, http.StatusOK, t, err)
}

func (c Controller) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	t, err := InsertTodo(r.Context(), c.DB, in)
	reply(w, http.StatusCreated, t, err)
}

func (c Controller) Update(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}
	t, err := UpdateTodo(r.Context(), c.DB, idParam(r), in)
	reply(w, http.StatusOK, t, err)
}

func (c Controller) Delete(w http.ResponseWriter, r *http.Request) {
	err := DeleteTodo(r.Context(), c.DB, idParam(r))
	reply(w, http.StatusNoContent, nil, err)
}
