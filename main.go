package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func withCorrectMimeType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	http.Handle("/workers/", http.StripPrefix("/workers/", http.FileServer(http.Dir("./web/workers"))))
	http.Handle("/books/", http.StripPrefix("/books/", http.FileServer(http.Dir("books"))))

	// Esta es la parte crítica
	http.Handle("/dist/", withCorrectMimeType(http.StripPrefix("/dist/", http.FileServer(http.Dir("./web/dist")))))

	http.Handle("/", http.FileServer(http.Dir("./web/static")))

	fmt.Println("Kindria started on http://localhost:4545")
	if err := http.ListenAndServe(":4545", nil); err != nil {
		log.Fatal("Could not start server", err)
	}
}

func serveChapter(w http.ResponseWriter, r *http.Request) {
	chapterNum := r.URL.Query().Get("n")
	if chapterNum == "" {
		http.Error(w, "Missing chapter number", http.StatusBadRequest)
	}

	epubPath := "./books/libro.epub"
	rdr, err := zip.OpenReader(epubPath)
	if err != nil {
		http.Error(w, "Could not unzip the epub file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer rdr.Close()

	var chapterFiles []string

	for _, f := range rdr.File {
		if strings.HasSuffix(f.Name, ".xhtml") || strings.HasSuffix(f.Name, ".html") {
			chapterFiles = append(chapterFiles, f.Name)
		}
	}

	index := 0
	fmt.Scanf(chapterNum, "d", &index)
	if index < 0 || index >= len(chapterFiles) {
		http.Error(w, "Could not find chapter", http.StatusInternalServerError)
		return
	}

	file := chapterFiles[index]
	for _, f := range rdr.File {
		if f.Name == file {
			rc, err := f.Open()
			if err != nil {
				http.Error(w, "Error openiing chapter", http.StatusInternalServerError)
			}

			defer rc.Close()

			content, _ := io.ReadAll(rc)
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
			return
		}
	}

}
