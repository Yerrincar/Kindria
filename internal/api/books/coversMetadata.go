package metadata

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
	 Logic:
		1. Check cover in the cache/database so we don't need to fecth again. (I need to add the save to cache|database)
		2. If not, extract local metadata: author, title, etc
		3. Search external api with that info, if we find the cover, return it and save it to the cache/database
		4. If the previous step fails, use the .jpeg/.jpg/.png file inside the .epub file
*/
func (p *Package) ProcessCover() (string, error) {
	coverApiPath, err := p.extractCoverFromApi()
	if err != nil {
		log.Printf("Eror trying to call extractCoverFromApi func: %v", err)
	}
	if coverApiPath != "" {
		return coverApiPath, nil
	}

	coverEpubPath, err := p.extractCoverFromEpub()
	fmt.Println("COVER EPUB:", coverEpubPath)
	if err != nil {
		log.Printf("Eror trying to call extractCoverFromEpub func: %v", err)
	}
	if coverEpubPath != "" {
		return coverEpubPath, nil
	}

	return "", nil
}

func (p *Package) extractCoverFromEpub() (string, error) {
	finalPath := ("./cache/covers/" + strings.TrimSpace(p.Metadata.Title) + ".jpg")
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

func (p *Package) extractCoverFromApi() (string, error) {
	// URL Covers IP https://covers.openlibrary.org/b/id/13540618-L.jpg
	finalPath := ("./cache/covers/" + strings.TrimSpace(p.Metadata.Title) + ".jpg")
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
