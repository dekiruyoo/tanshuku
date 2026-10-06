package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

var db *sql.DB

type ShortenRequest struct {
	URL string `json:"url"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Pong! Shortener is up and running!")
}

func generateShort() string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // Allowed chars
	var newLink strings.Builder
	for range 7 {
		newLink.WriteByte(charset[rand.Intn(len(charset))])
	}
	fmt.Println("New link generated: " + newLink.String())
	return newLink.String()
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	var existingCode string
	var tanshukukei string
	json.NewDecoder(r.Body).Decode(&req)
	success := false

	row := db.QueryRow("SELECT short_code FROM links WHERE original_url = $1", req.URL)

	err := row.Scan(&existingCode)

	if err == sql.ErrNoRows {
		for range 100 {
			tanshukukei = generateShort() // Japanese: contraction, shortened form, abbreviated form
			_, err := db.Exec("INSERT INTO links (original_url, short_code) VALUES ($1, $2)", req.URL, tanshukukei)

			if err == nil {
				success = true
				break
			}

			pqErr, isPqError := err.(*pq.Error)
			isCollision := isPqError && pqErr.Code == "23505"

			if !isCollision {
				http.Error(w, "error trying to save: ", http.StatusInternalServerError)
				log.Println(err)
				return
			}
		}
		if !success {
			http.Error(w, "You just lost the game!", http.StatusTeapot) // Reaching this error means the code attempted to fix link collision 100 times and failed all of them
			return
		}
		fmt.Fprintf(w, "Generated short URL for %s = %s\n", req.URL, tanshukukei)
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Println(err)
		return
	} else {
		fmt.Fprintf(w, "Received URL %s has already been shortened: %s\n", req.URL, existingCode)
		return
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	tanshukukei := r.PathValue("tanshukukei")
	var originalUrl string
	row := db.QueryRow("SELECT original_url FROM links WHERE short_code = $1", tanshukukei)

	err := row.Scan(&originalUrl)

	if err == sql.ErrNoRows {
		http.Error(w, "error: not found.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unknown error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	http.Redirect(w, r, originalUrl, http.StatusFound)
}

func main() {
	// Declare database connection with error treatment.
	var err error
	godotenv.Load()
	connStr := os.Getenv("DATABASE_URL")
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Routes. The port is for the end user to connect
	port := "8080"

	log.Printf("Starting server on port %s...\n", port)
	http.HandleFunc("GET /", helloHandler)
	http.HandleFunc("POST /shorten", shortenHandler)
	http.HandleFunc("GET /{tanshukukei}", redirectHandler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
