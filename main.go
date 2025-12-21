package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web/build")))

	http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir("./books"))))

	fmt.Println("Kindria running on http://localhost:4545")
	log.Fatal(http.ListenAndServe(":4545", nil))
}
