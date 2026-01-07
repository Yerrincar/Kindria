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
	_ "modernc.org/sqlite"
	"net/http"
	"net/url"
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

type jsonWrapper struct {
	NumFound int     `json:"numFound"`
	Docs     []OLDoc `json:"docs"`
}

type OLDoc struct {
	Cover_i int `json:"cover_i"`
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
	fileNameMap := make(map[string]bool)

	data, err := os.ReadDir(path)
	if err != nil {
		log.Fatal("Err while reading the books folder: ", err)
		return nil, err
	}

	insertedJson := make([]db.Book, len(data))

	fnSlice, err := h.Queries.SelectFileNames(ctx)
	if err != nil {
		log.Printf("Err trying to get all file names: %v", err)
		return nil, err
	}
	for _, f := range fnSlice {
		fileNameMap[f] = true
	}

	for _, e := range data {
		if fileNameMap[e.Name()] {
			continue
		}

		if e.IsDir() || !strings.HasSuffix(e.Name(), "epub") {
			continue
		}
		bookData, err := extractMetadata(e.Name())
		if err != nil {
			log.Printf("\nErr extracting data from book: %s | %v", e.Name(), err)
			continue
		}
		coverPath, err := bookData.ProcessCover()
		if err != nil {
			log.Printf("Error trying to get cover path: %v", err)
		}
		booksJson, err := h.Queries.InsertBooks(ctx, db.InsertBooksParams{
			Title:       bookData.Metadata.Title,
			Author:      bookData.Metadata.Author,
			Description: bookData.Metadata.Description,
			Genders:     bookData.Metadata.Genders,
			Language:    bookData.Metadata.Language,
			FileName:    bookData.BookFile,
			Bookpath:    coverPath,
		})
		if err != nil {
			return nil, err
		}
		insertedJson = append(insertedJson, booksJson...)
	}
	return insertedJson, nil
}

func extractMetadata(src string) (*Package, error) {
	var BookData Package
	initialPath := "./books/"
	r, err := zip.OpenReader(initialPath + src)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
		return nil, err
	}

	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".opf") {
			rc, err := f.Open()
			if err != nil {
				log.Printf("Err trying to access the .opf file: %v", err)
				continue
			}
			rcBytes, err := io.ReadAll(rc)
			if err != nil {
				log.Printf("Err trying to read the content of the .opf file: %v", err)
				continue
			}
			rc.Close()
			err = xml.Unmarshal(rcBytes, &BookData)
			if err != nil {
				log.Printf("Err parsing xml data: %v", err)
				continue
			}
			baseDir := path.Dir(f.Name)
			for _, m := range BookData.Manifest.Items {
				if m.Id == "cover" { //Could be a good idea to check something more robust than cover
					BookData.InternalCoverPath = path.Join(baseDir, m.Href)
					BookData.BookFile = src
					break
				}
			}
			break
		}
	}
	return &BookData, nil
}

func SearchOpenLibrary(title, author string) (int, error) {
	var o jsonWrapper

	u, err := url.Parse("https://openlibrary.org/search.json?title=reyes+de+la+tierra+salvaje&author=Nicholas+Eames&limit=2&offset=0")
	if err != nil {
		log.Printf("Error trying to parse base url: %v", err)
		return 0, err
	}
	q := u.Query()
	q.Set("title", title)
	q.Set("author", author)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	contact := os.Getenv("OLContact")
	req.Header.Set("User-Agent", "Kindria/0.1 (contact: "+contact+")")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&o)
	if err != nil {
		return 0, err
	}

	if o.NumFound == 0 {
		return 0, err
	}
	coverId := o.Docs[0].Cover_i
	if coverId > 0 {
		return coverId, nil
	}
	return 0, nil
}
