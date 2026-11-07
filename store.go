package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Turtle struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type turtleInput struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Controller struct {
	DB *pgxpool.Pool
}

func ListTurtles(ctx context.Context, db *pgxpool.Pool) ([]Turtle, error) {
	rows, err := db.Query(ctx, `SELECT id, name, age FROM turtles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Turtle{}
	for rows.Next() {
		var t Turtle
		if err := rows.Scan(&t.ID, &t.Name, &t.Age); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func FindTurtle(ctx context.Context, db *pgxpool.Pool, id int) (Turtle, error) {
	var t Turtle
	err := db.QueryRow(ctx, `SELECT id, name, age FROM turtles WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Age)
	return t, err
}

func InsertTurtle(ctx context.Context, db *pgxpool.Pool, name string, age int) (Turtle, error) {
	var t Turtle
	err := db.QueryRow(ctx,
		`INSERT INTO turtles (name, age) VALUES ($1, $2) RETURNING id, name, age`,
		name, age).Scan(&t.ID, &t.Name, &t.Age)
	return t, err
}

func UpdateTurtle(ctx context.Context, db *pgxpool.Pool, id int, name string, age int) (Turtle, error) {
	var t Turtle
	err := db.QueryRow(ctx,
		`UPDATE turtles SET name = $1, age = $2 WHERE id = $3 RETURNING id, name, age`,
		name, age, id).Scan(&t.ID, &t.Name, &t.Age)
	return t, err
}

func DeleteTurtle(ctx context.Context, db *pgxpool.Pool, id int) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM turtles WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (c Controller) Index(w http.ResponseWriter, r *http.Request) {
	rows, err := ListTurtles(r.Context(), c.DB)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (c Controller) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	t, err := FindTurtle(r.Context(), c.DB, id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (c Controller) Create(w http.ResponseWriter, r *http.Request) {
	var in turtleInput
	if err := decode(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if in.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	t, err := InsertTurtle(r.Context(), c.DB, in.Name, in.Age)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (c Controller) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var in turtleInput
	if err := decode(r, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if in.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	t, err := UpdateTurtle(r.Context(), c.DB, id, in.Name, in.Age)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (c Controller) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	n, err := DeleteTurtle(r.Context(), c.DB, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database"})
		return
	}
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
