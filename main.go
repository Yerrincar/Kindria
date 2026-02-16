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
	logFile, err := os.OpenFile("kindria.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}
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
	log.Printf("Opening DB")
	database, err := sql.Open("sqlite", "./books.db")
	if err != nil {
		log.Printf("Error opening database:  %v", err)
	}
	h := &metadata.Handler{Queries: db.New(database), DB: database, CM: metadata.NewCoverManager()}
	log.Printf("DB Open")

	log.Printf("Inserting books")
	_, err = h.InsertBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	log.Printf("Books inserted")
	log.Printf("Selecting books")
	books, err := h.SelectBooks()
	if err != nil {
		fmt.Printf("Error inserting books:  %v", err)
	}
	log.Printf("Books selected")
	go h.UpdateCacheCovers()

	log.Printf("Initializing TUI")
	p := tea.NewProgram(
		tui.InitialModel(books, h),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Kindria: %v", err)
	}
	CleanUpEscapeCode := "\x1b[2J\x1b[3J\x1b[H"
	fmt.Print(CleanUpEscapeCode)

	log.Printf("TUI Initialized")
}
