package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./web")))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Could not start server", err)
	}
}
