package pdfops

import (
	"fmt"
	"image/color"
	"os"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// RedactPDF finds every occurrence of textToRedact and places a black rectangle over it.
func RedactPDF(inFile, outFile, textToRedact string) error {
	matches, err := FindText(inFile, textToRedact)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("text %q not found in document", textToRedact)
	}

	if err := copyFile(inFile, outFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	for _, m := range matches {
		bboxW := m.X1 - m.X0
		bboxH := m.Y1 - m.Y0
		if bboxH < 4 {
			bboxH = 10
		}

		// Redaction margins (slightly larger to ensure full coverage)
		hMargin, vMargin := 2.0, 2.0
		coverW := int(bboxW + 2*hMargin + 0.5)
		coverH := int(bboxH + 2*vMargin + 0.5)
		
		// Solid black cover
		bg := color.RGBA{0, 0, 0, 255}
		coverFile, err := makeCover(coverW, coverH, bg)
		if err != nil {
			return fmt.Errorf("cover: %w", err)
		}
		
		coverDesc := fmt.Sprintf(
			"scalefactor:1 abs, position:tl, offset:%.2f %.2f, opacity:1, rotation:0",
			m.X0-hMargin, -(m.Y0 - vMargin),
		)
		wm, err := api.ImageWatermark(coverFile, coverDesc, true, false, types.POINTS)
		if err != nil {
			os.Remove(coverFile)
			return fmt.Errorf("redact wm: %w", err)
		}
		
		pageSel := []string{strconv.Itoa(m.Page)}
		if err := api.AddWatermarksFile(outFile, "", pageSel, wm, model.NewDefaultConfiguration()); err != nil {
			os.Remove(coverFile)
			return fmt.Errorf("apply redact: %w", err)
		}
		os.Remove(coverFile)
	}
	return nil
}
