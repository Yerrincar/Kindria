package reader

import (
	"fmt"
	"log"
	"rsc.io/pdf"
)

func Reader(filePath string) {
	r, err := pdf.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open the PDF file: %v", err)
	}

	for i := 0; i < r.NumPage(); i++ {
		page := r.Page(i)
		content := page.Content()
		for _, text := range content.Text {
			fmt.Print(text)
		}
		fmt.Println("\n--- End of page ---")
	}
}
