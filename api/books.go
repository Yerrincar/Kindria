package api

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Package struct {
	XMLName  xml.Name `xml:"package"`
	Metadata MetaData `xml:"metadata"`
	// Manifest Manifest `xml:"manifest"`
}

type MetaData struct {
	Author string `xml:"http://purl.org/dc/elements/1.1/ creator" json:"author"`
	Title  string `xml:"http://purl.org/dc/elements/1.1/ title" json:"title"`
}

//type Manifest struct {
//	Title string `xml:"http://purl.org/dc/elements/1.1/ title" json:"title"`
//}

func ServeJson(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		structs, err := serveStruct()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(structs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func serveStruct() ([]MetaData, error) {
	path := "./books/"
	data, err := os.ReadDir(path)
	booksJson := make([]MetaData, 0, len(data))
	if err != nil {
		log.Fatal("Err while reading the books folder: ", err)
	}
	for _, e := range data {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "epub") {
			bookData, err := extractMetadata(e.Name())
			if err != nil {
				log.Printf("\nErr extracting data from book: %s | %v", e.Name(), err)
			}
			booksJson = append(booksJson, bookData.Metadata)
		}
	}
	return booksJson, nil
}

func extractMetadata(src string) (data *Package, err error) {
	var BookData Package
	path := "./books/"
	r, err := zip.OpenReader(path + src)
	if err != nil {
		log.Fatalf("Err opening .epub file: %v", err)
	}

	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".opf") {
			rc, err := f.Open()
			if err != nil {
				log.Fatalf("Err trying to access the .opf file: %v", err)
			}
			rcBytes, err := io.ReadAll(rc)
			if err != nil {
				log.Fatalf("Err trying to read the content of the .opf file: %v", err)
			}
			err = xml.Unmarshal(rcBytes, &BookData)
			if err != nil {
				log.Fatalf("Err parsing xml data: %v", err)
			}
		}
	}
	return &BookData, nil
}
