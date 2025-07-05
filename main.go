package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/CloudyKit/jet/v6"
	"github.com/bmizerany/pat"
)

var jwtKey = []byte("my_secret_key")

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./templates"),
	jet.InDevelopmentMode(),
)

func main() {
	file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// To log to both file and console
	multiWriter := io.MultiWriter(file, os.Stdout)
	log.SetOutput(multiWriter)

	db, err := setupDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := pat.New()

	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		log.Fatal("Could not create uploads directory:", err)
	}
	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Get("/static/", http.StripPrefix("/static/", fs))
	mux.Get("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Routes
	mux.Get("/", http.HandlerFunc(menu))
	mux.Get("/login", http.HandlerFunc(showLoginPage)) // Page to show the login form
	mux.Post("/login", http.HandlerFunc(login))        // Handle login form submission
	mux.Post("/logout", http.HandlerFunc(logout))

	mux.Get("/admin", authMiddleware(http.HandlerFunc(admin)))
	mux.Post("/admin/add", authMiddleware(http.HandlerFunc(addItem)))
	mux.Post("/admin/edit/:id", authMiddleware(http.HandlerFunc(editItem)))
	mux.Post("/admin/delete/:id", authMiddleware(http.HandlerFunc(deleteItem)))
	mux.Post("/admin/discount", authMiddleware(http.HandlerFunc(applyDiscountToItem)))

	var handler http.Handler = mux
	handler = loggingMiddleware(handler)
	handler = tollboothMiddleware(handler)

	log.Println("Running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
