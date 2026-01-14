package main

import (
	"Kindria/internal/api/books"
	"Kindria/internal/db"
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

func main() {
	database, err := sql.Open("sqlite", "./books.db")
	if err != nil {
		log.Printf("Error opening database:  %v", err)
	}
	h := &metadata.Handler{Queries: db.New(database), DB: database, CM: metadata.NewCoverManager()}
	_, err = h.InsertBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	go h.UpdateCacheCovers()

	http.Handle("/", http.FileServer(http.Dir("./web/build")))
	http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir("./books"))))
	http.Handle("/covers/", http.StripPrefix("/covers/", http.FileServer(http.Dir("./cache/covers"))))

	http.HandleFunc("/api/books/getbooks", h.ServeJson)

	log.Println("Kindria running on http://localhost:4545")
	log.Fatal(http.ListenAndServe(":4545", nil))
}
