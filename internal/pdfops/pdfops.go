// Package pdfops wraps the PDF pipeline: count pages, render previews,
// find text bounding boxes, and replace text by overlaying.
//
// We deliberately shell out to poppler (pdftotext, pdftoppm) for extraction
// and rendering because pure-Go libs do not give reliable bbox info for
// arbitrary PDFs. Replacement uses a "redact + overlay" strategy: cover
// the old text region with a rectangle in the page background color, then
// stamp the new text using pdfcpu. This preserves layout for short
// substitutions; longer replacements may overflow.
//
// For production-grade font/color preservation, swap Replace with a
// MuPDF (mutool) or PyMuPDF microservice — see docs/REPLACEMENT.md.
package pdfops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Match is a single occurrence of the searched text on a page.
// Coordinates are in PDF points (1/72 inch), origin top-left,
// matching what pdftotext -bbox-layout emits.
type Match struct {
	Page  int     `json:"page"`
	X0    float64 `json:"x0"`
	Y0    float64 `json:"y0"`
	X1    float64 `json:"x1"`
	Y1    float64 `json:"y1"`
	Text  string  `json:"text"`
	PageW float64 `json:"pageW"`
	PageH float64 `json:"pageH"`
}

// PageCount uses pdftotext metadata or pdfinfo to count pages.
func PageCount(pdf string) (int, error) {
	out, err := exec.Command("pdfinfo", pdf).Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo: %w", err)
	}
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "Pages:") {
			v := strings.TrimSpace(strings.TrimPrefix(ln, "Pages:"))
			n, err := strconv.Atoi(v)
			if err != nil {
				return 0, err
			}
			return n, nil
		}
	}
	return 0, fmt.Errorf("page count not found")
}

// RenderPage renders one page to a PNG byte slice at 110 DPI — enough
// for a clear on-screen preview without bloating bandwidth.
//
// poppler's pdftoppm writes a numbered file per page rather than to
// stdout, so we route through a tempfile and read it back.
func RenderPage(pdf, page string) ([]byte, error) {
	n, err := strconv.Atoi(page)
	if err != nil || n < 1 {
		return nil, fmt.Errorf("bad page")
	}
	tmp, err := os.CreateTemp("", "pgrender-*")
	if err != nil {
		return nil, err
	}
	prefix := tmp.Name()
	tmp.Close()
	os.Remove(prefix)
	defer os.Remove(prefix + ".png")

	cmd := exec.Command("pdftoppm",
		"-png", "-r", "110",
		"-f", strconv.Itoa(n), "-l", strconv.Itoa(n),
		"-singlefile", pdf, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, stderr.String())
	}
	return os.ReadFile(prefix + ".png")
}
