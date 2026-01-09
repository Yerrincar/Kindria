package metadata

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func (p *Package) ProcessCover() (string, error) {
	initialPath := p.GoodQualityCover()
	apiBooks := make([]string, 0)
	finalPath := ("./cache/covers/" + strings.ReplaceAll("", "_", p.Metadata.Title) + ".jpg")
	_, err := os.Stat(finalPath)
	if err == nil {
		return "", nil
	}
	if initialPath == "" {
		apiBooks = append(apiBooks, p.Metadata.Title)
	}

	coverEpubPath, err := p.extractCoverFromEpub(initialPath)
	if err != nil {
		log.Printf("Eror trying to call extractCoverFromEpub func: %v", err)
	}
	if coverEpubPath != "" {
		return coverEpubPath, nil
	}

	for _, c := range apiBooks {
		coverApiPath, err := p.extractCoverFromApi(c)
		if err != nil {
			log.Printf("Eror trying to call extractCoverFromApi func: %v", err)
		}
		if coverApiPath != "" {
			return coverApiPath, nil
		}

	}

	return "", nil
}

func (p *Package) extractCoverFromEpub(name string) (string, error) {
	finalPath := ("./cache/covers/" + strings.ReplaceAll("", "_", name))
	path := "./books/"
	bookPath := p.InternalCoverPath
	z, err := zip.OpenReader(path + p.BookFile)
	if err != nil {
		log.Printf("Error opening .epub file: %v", err)
		return "", err
	}

	defer z.Close()

	for _, f := range z.File {
		if strings.EqualFold(f.Name, bookPath) {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			file, err := os.Create(finalPath)
			if err != nil {
				log.Printf("Err creating file for the cover: %v", err)
				return "", err
			}
			_, err = io.Copy(file, rc)
			if err != nil {
				return "", err
			}
			break
		}
	}
	return finalPath, err
}

func (p *Package) extractCoverFromApi(name string) (string, error) {
	finalPath := ("./cache/covers/" + strings.ReplaceAll("", "_", name) + ".jpg")
	cover_i, err := SearchOpenLibrary(p.Metadata.Title, p.Metadata.Author)
	if err != nil {
		log.Printf("Err getting cover_i for Covers API: %v", err)
		return "", err
	}
	if cover_i == 0 {
		return "", nil
	}
	u, err := url.JoinPath("https://covers.openlibrary.org/b/id/", strconv.Itoa(cover_i)+".jpg")
	if err != nil {
		log.Printf("Error trying to parse base url: %v", err)
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	contact := os.Getenv("OLContact")
	req.Header.Set("User-Agent", "Kindria/0.1 (contact: "+contact+")")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	file, err := os.Create(finalPath)
	if err != nil {
		log.Printf("Err creating file for the cover: %v", err)
		return "", err
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Error saving cover: %v", err)
		return "", err
	}
	return finalPath, nil
}
