package pdfops

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
)

// pdftotext -bbox-layout output schema (subset we care about).
type bboxDoc struct {
	XMLName xml.Name   `xml:"html"`
	Body    bboxBody   `xml:"body"`
}
type bboxBody struct {
	Doc bboxDocInner `xml:"doc"`
}
type bboxDocInner struct {
	Pages []bboxPage `xml:"page"`
}
type bboxPage struct {
	Width  float64    `xml:"width,attr"`
	Height float64    `xml:"height,attr"`
	Flows  []bboxFlow `xml:"flow"`
}
type bboxFlow struct {
	Blocks []bboxBlock `xml:"block"`
}
type bboxBlock struct {
	Lines []bboxLine `xml:"line"`
}
type bboxLine struct {
	XMin  float64    `xml:"xMin,attr"`
	YMin  float64    `xml:"yMin,attr"`
	XMax  float64    `xml:"xMax,attr"`
	YMax  float64    `xml:"yMax,attr"`
	Words []bboxWord `xml:"word"`
}
type bboxWord struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
	Text string  `xml:",chardata"`
}

// FindText searches every page for `needle` (case-insensitive) and
// returns one Match per occurrence with its bounding box.
//
// Strategy: pdftotext -bbox-layout emits per-word boxes. We walk each
// line, concatenate words, find substring matches, and union the boxes
// of the contributing words. Whitespace inside the needle is collapsed.
func FindText(pdf, needle string) ([]Match, error) {
	if strings.TrimSpace(needle) == "" {
		return nil, fmt.Errorf("empty needle")
	}
	cmd := exec.Command("pdftotext", "-bbox-layout", pdf, "-")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}
	var doc bboxDoc
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		return nil, fmt.Errorf("parse bbox xml: %w", err)
	}

	needleLow := strings.ToLower(strings.Join(strings.Fields(needle), " "))

	var matches []Match
	for pi, pg := range doc.Body.Doc.Pages {
		for _, fl := range pg.Flows {
			for _, blk := range fl.Blocks {
				for _, ln := range blk.Lines {
					matches = append(matches, scanLine(ln, needleLow, pi+1, pg.Width, pg.Height)...)
				}
			}
		}
	}
	return matches, nil
}

func scanLine(ln bboxLine, needle string, page int, pw, ph float64) []Match {
	if len(ln.Words) == 0 {
		return nil
	}
	// Build the lowercase line text and a parallel index from rune
	// offset back to the originating word so we can recover bboxes.
	var sb strings.Builder
	type span struct{ start, end int } // [start,end) char range per word
	spans := make([]span, len(ln.Words))
	for i, w := range ln.Words {
		if i > 0 {
			sb.WriteByte(' ')
		}
		spans[i].start = sb.Len()
		sb.WriteString(strings.ToLower(w.Text))
		spans[i].end = sb.Len()
	}
	hay := sb.String()

	var out []Match
	from := 0
	for {
		idx := strings.Index(hay[from:], needle)
		if idx < 0 {
			break
		}
		matchStart := from + idx
		matchEnd := matchStart + len(needle)

		// Find words that overlap [matchStart, matchEnd).
		var (
			x0, y0 = 1e18, 1e18
			x1, y1 = -1.0, -1.0
		)
		for i, s := range spans {
			if s.end <= matchStart || s.start >= matchEnd {
				continue
			}
			w := ln.Words[i]
			if w.XMin < x0 {
				x0 = w.XMin
			}
			if w.YMin < y0 {
				y0 = w.YMin
			}
			if w.XMax > x1 {
				x1 = w.XMax
			}
			if w.YMax > y1 {
				y1 = w.YMax
			}
		}
		if x1 > x0 && y1 > y0 {
			out = append(out, Match{
				Page:  page,
				X0:    x0, Y0: y0, X1: x1, Y1: y1,
				Text:  needle,
				PageW: pw, PageH: ph,
			})
		}
		from = matchEnd
		if from >= len(hay) {
			break
		}
	}
	return out
}
