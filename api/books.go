package api

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type Package struct {
	Metadata          MetaData `xml:"metadata" json:"metadata"`
	Manifest          Manifest `xml:"manifest" json:"-"`
	InternalCoverPath string   `json:"cover_path"`
	BookFile          string   `json:"book_name"`
}

type MetaData struct {
	Author string `xml:"http://purl.org/dc/elements/1.1/ creator" json:"author"`
	Title  string `xml:"http://purl.org/dc/elements/1.1/ title" json:"title"`
}
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Id   string `xml:"id,attr"`
	Href string `xml:"href,attr"`
}

func ServerCover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	path := "./books/"
	bookPath := r.URL.Query().Get("book")
	imagePath := r.URL.Query().Get("path")
	z, err := zip.OpenReader(path + bookPath)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
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

func serveStruct() ([]*Package, error) {
	path := "./books/"
	data, err := os.ReadDir(path)
	booksJson := make([]*Package, 0, len(data))
	if err != nil {
		log.Fatal("Err while reading the books folder: ", err)
	}
	for _, e := range data {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "epub") {
			bookData, err := extractMetadata(e.Name())
			if err != nil {
				log.Printf("\nErr extracting data from book: %s | %v", e.Name(), err)
			}
			booksJson = append(booksJson, bookData)
		}
	}
	return booksJson, nil
}

func extractMetadata(src string) (*Package, error) {
	var BookData Package
	initialPath := "./books/"
	r, err := zip.OpenReader(initialPath + src)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
	}

	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".opf") {
			rc, err := f.Open()
			if err != nil {
				log.Printf("Err trying to access the .opf file: %v", err)
			}
			rcBytes, err := io.ReadAll(rc)
			if err != nil {
				log.Printf("Err trying to read the content of the .opf file: %v", err)
			}
			err = xml.Unmarshal(rcBytes, &BookData)
			if err != nil {
				log.Printf("Err parsing xml data: %v", err)
			}
			baseDir := path.Dir(f.Name)
			for _, m := range BookData.Manifest.Items {
				if m.Id == "cover" {
					BookData.InternalCoverPath = path.Join(baseDir, m.Href)
					BookData.BookFile = src
				}
			}
		}
	}
	return &BookData, nil
}
