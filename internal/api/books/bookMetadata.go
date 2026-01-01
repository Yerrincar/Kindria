package metadata

import (
	"Kindria/internal/db"
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	_ "modernc.org/sqlite"
)

type Package struct {
	Metadata          MetaData `xml:"metadata" json:"metadata"`
	Manifest          Manifest `xml:"manifest" json:"-"`
	InternalCoverPath string   `json:"cover_path"`
	BookFile          string   `json:"book_name"`
}

type MetaData struct {
	Author      string `xml:"http://purl.org/dc/elements/1.1/ creator" json:"author"`
	Title       string `xml:"http://purl.org/dc/elements/1.1/ title" json:"title"`
	Description string `xml:"http://purl.org/dc/elements/1.1/ description" json:"description"`
	Genders     string `xml:"http://purl.org/dc/elements/1.1/ subject" json:"genders"`
	Language    string `xml:"http://purl.org/dc/elements/1.1/ language" json:"ln"`
}
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Id   string `xml:"id,attr"`
	Href string `xml:"href,attr"`
}

type Handler struct {
	Queries *db.Queries
	DB      *sql.DB
}

func (h *Handler) ServeJson(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		structs, err := h.Queries.ListBooks(ctx)
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

func (h *Handler) InsertBooks() ([]db.Book, error) {
	ctx := context.Background()
	path := "./books/"
	insertedJson := make([]db.Book, 0)
	fileNameMap := make(map[string]bool)

	data, err := os.ReadDir(path)
	if err != nil {
		log.Fatal("Err while reading the books folder: ", err)
	}

	fnSlice, err := h.Queries.SelectFileNames(ctx)
	if err != nil {
		log.Printf("Err trying to get all file names: %v", err)
	}
	for _, f := range fnSlice {
		fileNameMap[f] = true
	}

	for _, e := range data {
		if fileNameMap[e.Name()] {
			continue
		} else {
			if !e.IsDir() && strings.HasSuffix(e.Name(), "epub") {
				bookData, err := extractMetadata(e.Name())
				if err != nil {
					log.Printf("\nErr extracting data from book: %s | %v", e.Name(), err)
				}
				booksJson, err := h.Queries.InsertBooks(ctx, db.InsertBooksParams{
					Title:       bookData.Metadata.Title,
					Author:      bookData.Metadata.Author,
					Description: bookData.Metadata.Description,
					Genders:     bookData.Metadata.Genders,
					Language:    bookData.Metadata.Language,
					FileName:    bookData.BookFile,
					Bookpath:    bookData.InternalCoverPath,
				})
				if err != nil {
					return nil, err
				}
				insertedJson = append(insertedJson, booksJson...)
			}
		}
	}
	return insertedJson, nil
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
			rc.Close()
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
