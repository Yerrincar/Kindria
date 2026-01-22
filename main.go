package main

import (
	metadata "Kindria/internal/core/api/books"
	"Kindria/internal/core/db"
	"Kindria/internal/tui"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	//"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGUSR1)
		for range sigCh {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			log.Printf("SIGUSR1 goroutine dump:\n%s", buf[:n])
		}
	}()
	log.Printf("Abrimos DB")
	database, err := sql.Open("sqlite", "./books.db")
	if err != nil {
		log.Printf("Error opening database:  %v", err)
	}
	h := &metadata.Handler{Queries: db.New(database), DB: database, CM: metadata.NewCoverManager()}
	log.Printf("DB Abierta")
	//go h.UpdateCacheCovers()

	log.Printf("Insertamos libros")
	_, err = h.InsertBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	log.Printf("Libros Insertados")
	log.Printf("Seleccionamo libros")
	books, err := h.SelectBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	log.Printf("Libros Seleccionados")

	log.Printf("Iniciamos TUI")
	p := tea.NewProgram(tui.InitialModel(books, h), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Kindria: %v", err)
	}
	log.Printf("TUI Iniciada")
	//http.Handle("/", http.FileServer(http.Dir("./web/build")))
	//http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir(".././books"))))
	//http.Handle("/covers/", http.StripPrefix("/covers/", http.FileServer(http.Dir("./cache/covers"))))

	//http.HandleFunc("/api/books/getbooks", h.ServeJson)

	//log.Println("Kindria running on http://localhost:4545")
	//log.Fatal(http.ListenAndServe(":4545", nil))
}
