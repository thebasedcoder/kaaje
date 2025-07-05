package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type MenuItem struct {
	ID          int
	Name        string
	Description string
	Picture     string
	Price       float64
	Toppings    string
	Discount    float64
	Category    string // <-- Add this field
}

func setupDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./menu.db")
	if err != nil {
		return nil, err
	}

	// The 'category TEXT' field has been added to the SQL statement.
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS menu (id INTEGER PRIMARY KEY, name TEXT, description TEXT, picture TEXT, price REAL, toppings TEXT, discount REAL, category TEXT)")
	if err != nil {
		return nil, err
	}
	statement.Exec()
	return db, nil
}
