package main

import (
	"Kindria/internal/core/api/books"
	"Kindria/internal/core/db"
	"Kindria/internal/tui"
	"database/sql"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	//"net/http"
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

	books, err := h.SelectBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	go h.UpdateCacheCovers()

	p := tea.NewProgram(tui.InitialModel(books))
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Kindria: %v", err)
	}
	//http.Handle("/", http.FileServer(http.Dir("./web/build")))
	//http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir(".././books"))))
	//http.Handle("/covers/", http.StripPrefix("/covers/", http.FileServer(http.Dir("./cache/covers"))))

	//http.HandleFunc("/api/books/getbooks", h.ServeJson)

	//log.Println("Kindria running on http://localhost:4545")
	//log.Fatal(http.ListenAndServe(":4545", nil))
}
