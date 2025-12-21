package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
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

func main() {
	data, err := os.ReadDir("./../books/")
	booksJson := make([]MetaData, len(data))
	if err != nil {
		log.Fatal("Err while reading the books folder: ", err)
	}
	for _, e := range data {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "epub") {
			fmt.Print("\nNombre: ", e.Name())
			bookData, err := extractMetadata(e.Name())
			if err != nil {
				log.Fatalf("\nErr extracting data from book: %s | %v", e.Name(), err)
			}
			booksJson = append(booksJson, bookData.Metadata)
		}
	}
	bookDataJson, err := json.Marshal(booksJson)
	if err != nil {
		log.Fatalf("\nErr json data: %v", err)
	}
	fmt.Println("PRUEBA: ", string(bookDataJson))
	fmt.Println("Latest Book Title: ", booksJson[len(booksJson)-1].Title)
}

/* Los siguientes problemas que tengo que resolver son:
- Cambiar la función para devolver el el struct ya que será mejor enviarlo como tal y que el front haga el json.Marshal
- Luego, cuando le pase los datos en si, indicar que el handlefunc es un json también es importante
- Arreglar problema de capacidad para que solo pille los .epub o que sea y que aumente según se le añada, tercer parámetro de make....
*/

func extractMetadata(src string) (data *Package, err error) {
	var BookData Package
	path := "./../books/"
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
			fmt.Println(string(rcBytes))
			err = xml.Unmarshal(rcBytes, &BookData)
			if err != nil {
				log.Fatalf("Err parsing xml data: %v", err)
			}
			jsonBytes, _ := json.MarshalIndent(BookData.Metadata, "", " ")
			fmt.Println("MetaData: ", string(jsonBytes))
		}
	}
	return &BookData, nil
}
