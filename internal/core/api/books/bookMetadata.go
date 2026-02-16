package metadata

import (
	"Kindria/internal/core/db"
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
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
	Metadata          MetaData `xml:"metadata"`
	Manifest          Manifest `xml:"manifest"`
	Guide             Guide    `xml:"guide"`
	InternalCoverPath string   `json:"cover_path"`
	BookFile          string   `db:"file_name"`
	Rating            float64  `db:"rating"`
	Status            string   `db:"status"`
	ReadingDate       string   `db:"reading_date"`
}

type MetaData struct {
	Author      string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Title       string   `xml:"http://purl.org/dc/elements/1.1/ title"`
	Description string   `xml:"http://purl.org/dc/elements/1.1/ description"`
	Genres      []string `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Language    string   `xml:"http://purl.org/dc/elements/1.1/ language"`
	Metas       []Meta   `xml:"meta" json:"-"`
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

type Guide struct {
	References []Reference `xml:"reference"`
}

type Reference struct {
	Type  string `xml:"type,attr"`
	Href  string `xml:"href,attr"`
	Title string `xml:"title,attr"`
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
			Genres:      strings.Join(normalizeGenres(bookData.Metadata.Genres), ","),
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

func normalizeGenres(genres []string) []string {
	if len(genres) == 0 {
		return nil
	}
	out := make([]string, 0, len(genres))
	seen := make(map[string]struct{}, len(genres))
	for _, g := range genres {
		g = strings.TrimSpace(g)
		g = strings.Trim(g, ",")
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if _, ok := seen[g]; ok {
			continue
		}
		seen[g] = struct{}{}
		out = append(out, g)
	}
	return out
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
			coverGuideHref := ""
			for _, m := range BookData.Metadata.Metas {
				if m.Name == "cover" && m.Content != "" {
					coverID = m.Content
					break
				}
			}
			for _, ref := range BookData.Guide.References {
				if ref.Type == "cover" && ref.Href != "" {
					coverGuideHref = ref.Href
					break
				}
			}
			for _, m := range BookData.Manifest.Items {
				if (coverGuideHref != "" && m.Href == coverGuideHref) ||
					(coverID != "" && m.Id == coverID) ||
					strings.Contains(m.Properties, "cover-image") ||
					m.Id == "cover" {
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
			if BookData.InternalCoverPath == "" && coverGuideHref != "" {
				BookData.InternalCoverPath = path.Join(baseDir, coverGuideHref)
				BookData.BookFile = src
				ext := strings.ToLower(path.Ext(BookData.InternalCoverPath))
				if ext == ".xhtml" || ext == ".html" || ext == ".xml" {
					imgRel, err := resolveCoverFromXHTML(r, BookData.InternalCoverPath)
					if err == nil && imgRel != "" {
						BookData.InternalCoverPath = path.Join(path.Dir(BookData.InternalCoverPath), imgRel)
					}
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
		genresSlice := normalizeGenres(strings.Split(row.Genres, ","))
		p := &Package{
			Metadata: MetaData{
				Title:       row.Title,
				Author:      row.Author,
				Description: row.Description,
				Genres:      genresSlice,
				Language:    row.Language,
			},
			BookFile:    row.FileName,
			Status:      row.Status,
			ReadingDate: row.ReadingDate,
		}
		books = append(books, p)
	}
	return books, nil
}

func (h *Handler) SelectBookInfo() ([]*Package, error) {
	books := make([]*Package, 0)
	rows, err := h.Queries.ListBooks(context.Background())
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		var finalRating float64
		if row.Rating.Valid == false {
			finalRating = 0.0
		} else {
			finalRating = row.Rating.Float64
		}
		genresSlice := normalizeGenres(strings.Split(row.Genres, ","))
		p := &Package{
			Metadata: MetaData{
				Title:  row.Title,
				Author: row.Author,
				Genres: genresSlice,
			},
			BookFile:    row.FileName,
			Rating:      finalRating,
			Status:      row.Status,
			ReadingDate: row.ReadingDate,
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

func (h *Handler) UpdateBookStatus(status, fileName string) (string, error) {
	readingDate := ""
	if status == "Read" {
		readingDate = time.Now().Format("2006-01-02")
	}
	err := h.Queries.UpdateStatus(context.Background(), db.UpdateStatusParams{Status: status, ReadingDate: readingDate, FileName: fileName})
	if err != nil {
		return "", err
	}
	return readingDate, nil
}

func (h *Handler) UpdateBookRating(rating float64, fileName string) error {
	err := h.Queries.UpdateRating(context.Background(), db.UpdateRatingParams{
		Rating:   sql.NullFloat64{Float64: rating, Valid: true},
		FileName: fileName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) CheckBookExist(filename string) (int64, error) {
	exists, err := h.Queries.CheckBookExists(context.Background(), filename)
	if err != nil {
		return 0, err
	}
	return exists, nil
}

func (h *Handler) EnsureReadingDateColumn() error {
	rows, err := h.DB.Query("PRAGMA table_info(books)")
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		cid        int
		name       string
		colType    string
		notNull    int
		defaultV   sql.NullString
		primaryKey int
	)
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primaryKey); err != nil {
			return err
		}
		if name == "reading_date" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = h.DB.Exec("ALTER TABLE books ADD reading_date TEXT NOT NULL DEFAULT ''")
	if err != nil {
		return fmt.Errorf("alter books add reading_date: %w", err)
	}
	return nil
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
