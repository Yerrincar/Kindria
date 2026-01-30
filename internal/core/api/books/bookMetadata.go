package metadata

import (
	"Kindria/internal/core/db"
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

type CoverManager struct {
	coversQueue chan *Package
}

func NewCoverManager() *CoverManager {
	return &CoverManager{
		coversQueue: make(chan *Package, 128),
	}
}

type Package struct {
	Metadata          MetaData `xml:"metadata" json:"metadata"`
	Manifest          Manifest `xml:"manifest" json:"-"`
	InternalCoverPath string   `json:"cover_path"`
	BookFile          string   `db:"file_name" json:"book_name"`
}

type MetaData struct {
	Author      string `xml:"http://purl.org/dc/elements/1.1/ creator" json:"author"`
	Title       string `xml:"http://purl.org/dc/elements/1.1/ title" json:"title"`
	Description string `xml:"http://purl.org/dc/elements/1.1/ description" json:"description"`
	Genders     string `xml:"http://purl.org/dc/elements/1.1/ subject" json:"genders"`
	Language    string `xml:"http://purl.org/dc/elements/1.1/ language" json:"ln"`
	Metas       []Meta `xml:"meta" json:"-"`
}

type Meta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}
type Manifest struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Id         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	Properties string `xml:"properties,attr"`
}

type Handler struct {
	Queries *db.Queries
	DB      *sql.DB
	CM      *CoverManager
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
		coverPath, err := h.CM.ProcessCover(bookData)
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
			coverID := ""
			for _, m := range BookData.Metadata.Metas {
				if m.Name == "cover" && m.Content != "" {
					coverID = m.Content
					break
				}
			}
			for _, m := range BookData.Manifest.Items {
				if (coverID != "" && m.Id == coverID) || strings.Contains(m.Properties, "cover-image") || m.Id == "cover" {
					BookData.InternalCoverPath = path.Join(baseDir, m.Href)
					BookData.BookFile = src
					ext := strings.ToLower(path.Ext(BookData.InternalCoverPath))
					if ext == ".xhtml" || ext == ".html" || ext == ".xml" {
						imgRel, err := resolveCoverFromXHTML(r, BookData.InternalCoverPath)
						if err == nil && imgRel != "" {
							BookData.InternalCoverPath = path.Join(path.Dir(BookData.InternalCoverPath), imgRel)
						}
					}
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

func (h *Handler) SelectBooks() ([]*Package, error) {
	books := make([]*Package, 0)
	rows, err := h.Queries.SelectAllBooks(context.Background())
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		p := &Package{
			Metadata: MetaData{
				Title:       row.Title,
				Author:      row.Author,
				Description: row.Description,
				Genders:     row.Genders,
				Language:    row.Language,
			},
			BookFile: row.FileName,
		}
		books = append(books, p)
	}
	return books, nil
}

func (h *Handler) SelectBookPath(bookFile string) (string, error) {
	path, err := h.Queries.SelectBookPath(context.Background(), bookFile)
	if err != nil {
		return "", err
	}
	return path, nil
}

func resolveCoverFromXHTML(r *zip.ReadCloser, href string) (string, error) {
	f, err := findZipFile(r, href)
	if err != nil {
		return "", err
	}

	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if strings.EqualFold(se.Name.Local, "img") {
			for _, a := range se.Attr {
				if strings.EqualFold(a.Name.Local, "src") && a.Value != "" {
					return a.Value, nil
				}
			}
		}
	}
}

func findZipFile(r *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, os.ErrNotExist
}
