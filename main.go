package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.Handle("/worker/", http.StripPrefix("/worker/", http.FileServer(http.Dir("web/worker"))))
	http.Handle("/libros/", http.StripPrefix("/libros/", http.FileServer(http.Dir("libros"))))
	if err := http.ListenAndServe(":4545", nil); err != nil {
		log.Fatal("Could not start server", err)
	}
	fmt.Print("Kindria started on localhost:4545")
}
