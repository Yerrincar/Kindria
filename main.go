package main

import (
	"Kindria/api"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web/build")))

	http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir("./books"))))
	http.HandleFunc("/api/getbooks", api.ServeJson)
	http.HandleFunc("/api/getCovers", api.ServerCover)
	/* Fetching Done. Now I need to:
	Add style
	Learn how to save the book metadata into a database so the library loads instantly
	Fix Reader
	Kindle Detection Logic
	*/
	fmt.Println("Kindria running on http://localhost:4545")
	log.Fatal(http.ListenAndServe(":4545", nil))
}
