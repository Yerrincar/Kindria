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
	dimensionCap := 2.0 / 3.0
	uniqueColors := 5
	winner := ""
	bestScore := 0
	possibleCovers := make([]*zip.File, 0)
	r, err := zip.OpenReader(bookPath)
	if err != nil {
		log.Printf("Err opening .epub file: %v", err)
		return ""
	}
	defer r.Close()

	for _, z := range r.File {
		if strings.HasSuffix(z.Name, ".jpg") || strings.HasSuffix(z.Name, "jpeg") {
			rc, err := z.Open()
			if err != nil {
				return ""
			}
			imageCover, err := jpeg.DecodeConfig(rc)
			rc.Close()
			if err != nil {
				return ""
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
		currentScore := 0
		rc, err := c.Open()
		if err != nil {
			return ""
		}
		coverDecoding, err := jpeg.Decode(rc)
		rc.Close()
		if err != nil {
			return ""
		}
		currentScore = coverDecoding.Bounds().Dx() * coverDecoding.Bounds().Dy()
		if strings.Contains(c.Name, "cover") {
			currentScore *= 2
		}
		if float64(coverDecoding.Bounds().Dx())/float64(coverDecoding.Bounds().Dy()) == dimensionCap {
			currentScore *= 2
		}

		colorMap := make(map[color.Color]int)
		for i := coverDecoding.Bounds().Min.X; i < coverDecoding.Bounds().Max.X; i += 50 {
			for j := coverDecoding.Bounds().Min.Y; j < coverDecoding.Bounds().Max.Y; j += 50 {
				colorMap[coverDecoding.At(i, j)] += 1
			}
		}
		if len(colorMap) >= uniqueColors {
			currentScore += len(colorMap)
		}

		if currentScore > bestScore {
			bestScore = currentScore
			winner = c.Name
		}

	}
	return winner
}
