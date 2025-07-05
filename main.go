package main

import (
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/bmizerany/pat"
)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./templates"),
	jet.InDevelopmentMode(),
)

func main() {
	db, err := setupDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := pat.New()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Get("/static/", http.StripPrefix("/static/", fs))

	// Routes
	mux.Get("/", http.HandlerFunc(menu))
	mux.Get("/admin", http.HandlerFunc(admin))
	mux.Post("/admin/login", http.HandlerFunc(login))
	mux.Post("/admin/add", http.HandlerFunc(addItem))
	mux.Post("/admin/edit/:id", http.HandlerFunc(editItem))
	mux.Post("/admin/delete/:id", http.HandlerFunc(deleteItem))
	mux.Post("/admin/discount", http.HandlerFunc(addDiscount))

	log.Println("سرور در حال اجرا روی پورت 8080 است")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
