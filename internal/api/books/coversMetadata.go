package metadata

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

/*
	 Logic:
		1. Check cover in the cache/database so we don't need to fecth again. (I need to add the save to cache|database)
		2. If not, extract local metadata: author, title, etc
		3. Search external api with that info, if we find the cover, return it and save it to the cache/database
		4. If the previous step fails, use the .jpeg/.jpg/.png file inside the .epub file
*/

func ServerCover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	path := "./books/"
	bookPath := r.URL.Query().Get("book")
	fmt.Print("bookPath: ", bookPath)
	imagePath := r.URL.Query().Get("path")
	z, err := zip.OpenReader(path + bookPath)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
		return
	}
	defer z.Close()

	for _, f := range z.File {
		if strings.EqualFold(f.Name, imagePath) {
			rc, err := f.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			_, err = io.Copy(w, rc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
}
