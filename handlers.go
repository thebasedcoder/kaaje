package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/golang-jwt/jwt/v5"

	"github.com/didip/tollbooth/v7"
)

// Claims struct will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields like expiry.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// This handler prepares data for your public menu page.
func menu(w http.ResponseWriter, r *http.Request) {
	db, err := setupDB()
	if err != nil {
		log.Println("Database connection error:", err)
		http.Error(w, "Server Error", 500)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, description, picture, price, toppings, discount, category FROM menu")
	if err != nil {
		log.Println("Database query error:", err)
		http.Error(w, "Server Error", 500)
		return
	}
	defer rows.Close()

	// 1. IMPORTANT: Initialize slices of the correct struct type.
	var pizzas, burgers, sandwiches []MenuItem

	// 2. Loop through all rows from the database.
	for rows.Next() {
		item := MenuItem{}
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Picture, &item.Price, &item.Toppings, &item.Discount, &item.Category)
		if err != nil {
			log.Println("Database scan error:", err)
			continue
		}

		// 3. Use a switch to append the FULL item struct to the correct slice.
		switch item.Category {
		case "pizza":
			pizzas = append(pizzas, item)
		case "burger":
			burgers = append(burgers, item)
		case "sandwich":
			sandwiches = append(sandwiches, item)
		}
	}

	// 4. Pass the correctly populated slices to the template.
	vars := make(jet.VarMap)
	vars.Set("pizzas", pizzas)
	vars.Set("burgers", burgers)
	vars.Set("sandwiches", sandwiches)

	view, err := views.GetTemplate("menu.jet") // Or whatever you named your file
	if err != nil {
		log.Println("Template getting error:", err)
		http.Error(w, "Server Error", 500)
		return
	}

	if err := view.Execute(w, vars, nil); err != nil {
		log.Println("Error executing template:", err) // This is where your error occurs
	}
}

func admin(w http.ResponseWriter, r *http.Request) {
	db, _ := setupDB()
	defer db.Close()

	rows, err := db.Query("SELECT id, name, description, picture, price, toppings, discount, category FROM menu ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}
	defer rows.Close()

	// THIS MUST BE of type []MenuItem
	var items []MenuItem

	for rows.Next() {
		// THIS MUST create a new MenuItem struct
		item := MenuItem{}

		// THIS MUST scan into the struct's fields
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Picture, &item.Price, &item.Toppings, &item.Discount, &item.Category)
		if err != nil {
			log.Println("DB Scan Error:", err)
			continue
		}

		// THIS MUST append the whole struct
		items = append(items, item)
	}

	vars := make(jet.VarMap)
	vars.Set("items", items)

	view, err := views.GetTemplate("admin.jet")
	if err != nil {
		http.Error(w, "Server Error", 500)
		return
	}

	if err := view.Execute(w, vars, nil); err != nil {
		log.Println("Template Execution Error:", err)
	}
}

// showLoginPage renders the login page template.
func showLoginPage(w http.ResponseWriter, r *http.Request) {
	view, err := views.GetTemplate("login.jet")
	if err != nil {
		http.Error(w, "Could not load login page", http.StatusInternalServerError)
		return
	}
	view.Execute(w, nil, nil)
}

func login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	// In a real app, you would check the username and password against the database.
	// Here, we'll just use static credentials.
	if username != "admin" || password != "password" {
		http.Error(w, "نام کاربری یا رمز عبور اشتباه است", http.StatusUnauthorized)
		return
	}

	// Declare the expiration time of the token (e.g., 24 hours).
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create the JWT claims, which includes the username and expiry time.
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string.
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	// Set the token as a cookie.
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
		Path:    "/", // Make the cookie available on all pages
	})

	// Redirect to the admin page on successful login.
	http.Redirect(w, r, "/admin", http.StatusFound)
}

// logout clears the authentication cookie.
func logout(w http.ResponseWriter, r *http.Request) {
	// Set the cookie with an expiration time in the past to delete it.
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
		Path:    "/",
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func addItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}

	// 1. Handle File Upload
	r.ParseMultipartForm(10 << 20) // 10 MB
	file, handler, err := r.FormFile("picture")
	if err != nil {
		http.Error(w, "Error Retrieving the File", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a unique filename and save the file
	uniqueFileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), handler.Filename)
	dst, err := os.Create(filepath.Join("./uploads", uniqueFileName))
	if err != nil {
		http.Error(w, "Error Saving the File", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	imagePath := "/uploads/" + uniqueFileName

	// 2. Parse other form fields
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	discount, _ := strconv.ParseFloat(r.FormValue("discount"), 64)

	// 3. Insert into database
	db, _ := setupDB()
	defer db.Close()
	statement, _ := db.Prepare("INSERT INTO menu (name, description, picture, price, toppings, discount, category) VALUES (?, ?, ?, ?, ?, ?, ?)")
	defer statement.Close()
	statement.Exec(r.FormValue("name"), r.FormValue("description"), imagePath, price, r.FormValue("toppings"), discount, r.FormValue("category"))

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func editItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}

	// Use ParseMultipartForm to handle both files and form values.
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max memory
		http.Error(w, "Invalid Form Data", http.StatusBadRequest)
		return
	}

	// --- Get and Validate ID ---
	id := r.URL.Query().Get(":id")
	if _, err := strconv.Atoi(id); err != nil {
		http.Error(w, "Invalid or missing item ID", http.StatusBadRequest)
		return
	}

	// --- Get and Validate Form Values ---
	name := r.FormValue("name")
	category := r.FormValue("category")
	if name == "" || category == "" {
		http.Error(w, "Name and Category are required", http.StatusBadRequest)
		return
	}
	// Correctly handle parsing errors for price and discount
	price, err := strconv.ParseFloat(r.FormValue("price"), 64)
	if err != nil {
		http.Error(w, "Invalid price format", http.StatusBadRequest)
		return
	}
	discount, err := strconv.ParseFloat(r.FormValue("discount"), 64)
	if err != nil {
		discount = 0 // Default to 0 if missing or invalid
	}

	// --- Connect to DB ---
	db, err := setupDB()
	if err != nil {
		log.Println("Database connection error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// --- Handle Optional File Upload ---
	var picturePath string
	file, handler, err := r.FormFile("picture")
	// `err == http.ErrMissingFile` means no new file was uploaded, which is okay.
	if err != nil && err != http.ErrMissingFile {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}

	if file != nil { // A new file was uploaded
		defer file.Close()

		// 1. Delete the old picture
		var oldPicturePath string
		db.QueryRow("SELECT picture FROM menu WHERE id=?", id).Scan(&oldPicturePath)
		if oldPicturePath != "" {
			os.Remove(filepath.Join(".", oldPicturePath))
		}

		// 2. Save the new picture
		uniqueFileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), handler.Filename)
		dst, _ := os.Create(filepath.Join("./uploads", uniqueFileName))
		defer dst.Close()
		io.Copy(dst, file)
		picturePath = "/uploads/" + uniqueFileName

		// 3. Update the picture path in the database
		stmt, _ := db.Prepare("UPDATE menu SET picture=? WHERE id=?")
		stmt.Exec(picturePath, id)
		stmt.Close()
	}

	// --- Update Other Item Details ---
	statement, err := db.Prepare("UPDATE menu SET name=?, description=?, price=?, toppings=?, discount=?, category=? WHERE id=?")
	if err != nil {
		log.Println("Database prepare error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	_, err = statement.Exec(name, r.FormValue("description"), price, r.FormValue("toppings"), discount, category, id)
	if err != nil {
		log.Println("Database execution error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}

	// Get the ID from the URL parameter instead of the form body.
	id := r.URL.Query().Get(":id")
	if _, err := strconv.Atoi(id); err != nil {
		http.Error(w, "Invalid or missing item ID", http.StatusBadRequest)
		return
	}

	db, err := setupDB()
	if err != nil {
		log.Println("Database connection error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Optional but recommended: Delete the associated image file.
	var picturePath string
	// It's good practice to check for an error here.
	err = db.QueryRow("SELECT picture FROM menu WHERE id=?", id).Scan(&picturePath)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Could not query picture for ID %s: %v", id, err)
	}
	if picturePath != "" {
		// remove the leading slash for local file system path
		os.Remove(filepath.Join(".", picturePath))
	}

	// Prepare and execute the DELETE statement with error handling.
	statement, err := db.Prepare("DELETE FROM menu WHERE id=?")
	if err != nil {
		log.Println("Database prepare error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	if _, err := statement.Exec(id); err != nil {
		log.Println("Database execution error:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func applyDiscountToItem(w http.ResponseWriter, r *http.Request) {
	// 1. Establish DB connection and handle errors
	db, err := setupDB()
	if err != nil {
		log.Println("Failed to connect to database:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// 2. Get and validate the item ID from the URL
	idStr := r.URL.Query().Get("id") // Assumes /discount?id=123
	if _, err := strconv.Atoi(idStr); err != nil {
		http.Error(w, "Invalid or missing item ID", http.StatusBadRequest)
		return
	}

	// 3. Parse and validate the discount value from the form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid Form Data", http.StatusBadRequest)
		return
	}
	discount, err := strconv.ParseFloat(r.FormValue("discount"), 64)
	if err != nil || discount < 0 || discount > 100 {
		http.Error(w, "Invalid discount value. Must be between 0 and 100.", http.StatusBadRequest)
		return
	}

	// 4. Prepare the UPDATE statement with a WHERE clause
	statement, err := db.Prepare("UPDATE menu SET discount=? WHERE id=?")
	if err != nil {
		log.Println("Failed to prepare statement:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer statement.Close()

	// 5. Execute the statement and handle errors
	_, err = statement.Exec(discount, idStr)
	if err != nil {
		log.Println("Failed to execute statement:", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// authMiddleware checks for a valid JWT before allowing access to a route.
func authMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the cookie from the request.
		c, err := r.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				// If the cookie is not set, redirect to the login page.
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			// For any other error, return a bad request status.
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Get the JWT string from the cookie.
		tokenString := c.Value

		// Initialize a new instance of `Claims`.
		claims := &Claims{}

		// Parse the JWT string and store the result in `claims`.
		// Note that we are passing the key in which was used to sign the token.
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			// If the token is invalid, redirect to the login page.
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// If the token is valid, call the next handler in the chain.
		next.ServeHTTP(w, r)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request details
		log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

func tollboothMiddleware(next http.Handler) http.Handler {
	// Create a limiter that allows 10 requests per second.
	lmt := tollbooth.NewLimiter(10, nil)

	lmt.SetMessage("شما بیش از حد مجاز درخواست ارسال کرده‌اید.")
	lmt.SetMessageContentType("text/plain; charset=utf-8")
	lmt.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Rate limit exceeded for %s", r.RemoteAddr)
	})

	// CORRECTED: Use LimitHandler and pass the handler directly.
	return tollbooth.LimitHandler(lmt, next)
}
