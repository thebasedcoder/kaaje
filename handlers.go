package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/CloudyKit/jet/v6"
)

func menu(w http.ResponseWriter, r *http.Request) {
	db, _ := setupDB()
	rows, _ := db.Query("SELECT id, name, description, picture, price, toppings, discount FROM menu")
	var items []MenuItem
	for rows.Next() {
		item := MenuItem{}
		rows.Scan(&item.ID, &item.Name, &item.Description, &item.Picture, &item.Price, &item.Toppings, &item.Discount)
		items = append(items, item)
	}
	rows.Close()

	vars := make(jet.VarMap)
	vars.Set("items", items)

	view, err := views.GetTemplate("menu.jet")
	if err != nil {
		log.Println("Unexpected template err:", err.Error())
	}
	view.Execute(w, vars, nil)
}

func admin(w http.ResponseWriter, r *http.Request) {
	db, _ := setupDB()
	rows, _ := db.Query("SELECT id, name, description, picture, price, toppings, discount FROM menu")
	var items []MenuItem
	for rows.Next() {
		item := MenuItem{}
		rows.Scan(&item.ID, &item.Name, &item.Description, &item.Picture, &item.Price, &item.Toppings, &item.Discount)
		items = append(items, item)
	}
	rows.Close()

	vars := make(jet.VarMap)
	vars.Set("items", items)

	view, err := views.GetTemplate("admin.jet")
	if err != nil {
		log.Println("Unexpected template err:", err.Error())
	}
	view.Execute(w, vars, nil)
}

func login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.FormValue("username") == "admin" && r.FormValue("password") == "password" {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func addItem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db, _ := setupDB()
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	statement, _ := db.Prepare("INSERT INTO menu (name, description, picture, price, toppings) VALUES (?, ?, ?, ?, ?)")
	statement.Exec(r.FormValue("name"), r.FormValue("description"), r.FormValue("picture"), price, r.FormValue("toppings"))
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func editItem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db, _ := setupDB()
	id := r.URL.Query().Get(":id")
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	statement, _ := db.Prepare("UPDATE menu SET name=?, description=?, picture=?, price=?, toppings=? WHERE id=?")
	statement.Exec(r.FormValue("name"), r.FormValue("description"), r.FormValue("picture"), price, r.FormValue("toppings"), id)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	db, _ := setupDB()
	id := r.URL.Query().Get(":id")
	statement, _ := db.Prepare("DELETE FROM menu WHERE id=?")
	statement.Exec(id)
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func addDiscount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	db, _ := setupDB()
	discount, _ := strconv.ParseFloat(r.FormValue("discount"), 64)
	statement, _ := db.Prepare("UPDATE menu SET discount=?")
	statement.Exec(discount)
	http.Redirect(w, r, "/admin", http.StatusFound)
}
