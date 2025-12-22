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
	http.HandleFunc("/library", api.ServeJson)
	//API endpoint done, now the fron must fetch the data from it

	fmt.Println("Kindria running on http://localhost:4545")
	log.Fatal(http.ListenAndServe(":4545", nil))
}
