package metadata

import (
	"archive/zip"
	"image/color"
	"image/jpeg"
	"log"
	"strings"
)

func (p *Package) QualityScoreCover() (finalPath string) {
	bookPath := ("./books/" + p.BookFile)
	dimensionCap := 0.666
	score := make(map[string]int)
	r, err := zip.OpenReader(bookPath)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
		return err.Error()
	}
	defer r.Close()

	for _, z := range r.File {
		if strings.HasSuffix(z.Name, ".jpg") {
			rc, err := z.Open()
			if err != nil {
				return err.Error()
			}
			imageCover, err := jpeg.DecodeConfig(rc)
			if err != nil {
				return err.Error()
			}

			if float64(imageCover.Width)/float64(imageCover.Height) < dimensionCap {
				score[z.Name] += -20
			}

		}
	}
	return ""
}
