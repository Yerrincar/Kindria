package metadata

import (
	"archive/zip"
	"image/color"
	"image/jpeg"
	"log"
	"strings"
)

func (p *Package) GoodQualityCover() (finalPath string) {
	bookPath := ("./books/" + p.BookFile)
	dimensionCap := 0.666
	uniqueColors := 5
	finalPath = ""
	possibleCovers := make([]*zip.File, 0)
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
			rc.Close()
			if err != nil {
				return err.Error()
			}
			switch {
			case imageCover.Width <= 400 || imageCover.Height <= 600:
				continue
			case float64(imageCover.Width)/float64(imageCover.Height) > dimensionCap:
				continue
			case imageCover.Width == imageCover.Height:
				continue
			default:
				possibleCovers = append(possibleCovers, z)
			}
		}
	}

	for _, c := range possibleCovers {
		rc, err := c.Open()
		if err != nil {
			return err.Error()
		}
		coverDecoding, err := jpeg.Decode(rc)
		rc.Close()
		if err != nil {
			return err.Error()
		}

		colorMap := make(map[color.Color]int)
		for i := coverDecoding.Bounds().Min.X; i < coverDecoding.Bounds().Max.X; i += 100 {
			for j := coverDecoding.Bounds().Min.Y; j < coverDecoding.Bounds().Max.Y; j += 100 {
				colorMap[coverDecoding.At(i, j)] += 1
			}
		}
		if len(colorMap) >= uniqueColors {
			finalPath = c.Name
			return finalPath
		}
	}
	return finalPath
}
