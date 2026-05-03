package pdfops

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Mode tunes the replacement strategy for the source PDF type.
//   ModeVector  — typed/generated PDFs. Extracts font/size/color from the
//                 content stream; tight white-out cover.
//   ModeScanned — camera-captured / scanner+OCR PDFs. Skips font extraction
//                 (the visible text is raster pixels — there's no font to
//                 read); uses a wider white-out cover to absorb OCR
//                 alignment slop; renders the new text in Helvetica.
const (
	ModeVector  = "vector"
	ModeScanned = "scanned"
)

// Options refines replacement behaviour.
//   Mode        — "vector" (default) | "scanned" (camera/OCR PDF; wider cover, no font extraction)
//   ForceBold   — render the new text as bold regardless of detected weight
//   ForceItalic — render the new text as italic/oblique regardless of detection
type Options struct {
	Mode        string
	ForceBold   bool
	ForceItalic bool
}

// ReplaceWithOpts is the option-rich entry point. Replace() keeps the
// older positional signature for backward compatibility.
func ReplaceWithOpts(in, out, oldText, newText string, pages []int, opts Options) error {
	return replaceCore(in, out, oldText, newText, pages, opts)
}

// Replace finds every occurrence of `oldText`, restricted to the given
// pages (1-based; empty = all pages), and overlays each one with `newText`.
//
// Strategy:
//   1. Copy input → output.
//   2. Read the original PDF's content stream once to recover the font
//      name, point size, and fill color in effect at each match. (See
//      extract_style.go.)  This makes the replacement visually match the
//      surrounding text — same size, same color, similar font family.
//   3. For each match:
//        a. Stamp a white rectangle large enough to cover the old text.
//        b. Stamp the new text on top, using the recovered font/size/color
//           (with bbox-derived size as a fallback).
//   4. If the new text is wider than the old bbox, the font is shrunk
//      proportionally so it doesn't overflow into adjacent content.
//
// Limitations remain documented in docs/REPLACEMENT.md.
func Replace(in, out, oldText, newText string, pages []int, mode string) error {
	return replaceCore(in, out, oldText, newText, pages, Options{Mode: mode})
}

func replaceCore(in, out, oldText, newText string, pages []int, opts Options) error {
	mode := opts.Mode
	if mode != ModeScanned {
		mode = ModeVector
	}
	matches, err := FindText(in, oldText)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("text %q not found in document", oldText)
	}
	if len(pages) > 0 {
		set := make(map[int]bool, len(pages))
		for _, p := range pages {
			set[p] = true
		}
		filtered := matches[:0]
		for _, m := range matches {
			if set[m.Page] {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
	}
	if len(matches) == 0 {
		return fmt.Errorf("text %q not found on the selected pages", oldText)
	}

	// In scanned mode there's no source font/size/color to extract — the
	// visible text is raster pixels. Skip the parser to save a pass.
	var styles map[int]Style
	if mode == ModeVector {
		styles, _ = ExtractStyles(in, oldText)
	}

	if err := copyFile(in, out); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// One sampler per replace, used to pick the background color the
	// white-out cover should match. Rendering is cached per-page.
	sampler := newBGSampler(in)

	for _, m := range matches {
		st := styles[m.Page]
		if err := stampMatch(out, m, oldText, newText, st, mode, sampler, opts); err != nil {
			return fmt.Errorf("stamp page %d: %w", m.Page, err)
		}
	}
	return nil
}

func stampMatch(file string, m Match, oldText, newText string, st Style, mode string, sampler *bgSampler, opts Options) error {
	bboxW := m.X1 - m.X0
	bboxH := m.Y1 - m.Y0
	if bboxH < 4 {
		bboxH = 10
	}

	// Pick a font size. The bbox-derived value is the safety net; if the
	// content-stream extraction agrees within ~2× we trust it (it's more
	// accurate). If it disagrees wildly — e.g., a PDF that uses Tf size=1
	// with a scale matrix in some less-common form — we fall back to the
	// bbox so we never produce a microscopic or gigantic replacement.
	bboxPts := bboxH / 0.92
	fontPtsF := bboxPts
	if st.HaveSize && st.Size > 0 {
		ratio := st.Size / bboxPts
		if ratio >= 0.55 && ratio <= 1.8 {
			fontPtsF = st.Size
		}
	}

	// If the new text would be wider than the old bbox at this size,
	// shrink the font so it fits.  Helvetica avg char width ≈ 0.50 em.
	avgCharW := 0.50
	expectedW := float64(len(newText)) * avgCharW * fontPtsF
	if expectedW > bboxW && bboxW > 0 {
		shrink := bboxW / expectedW
		if shrink < 0.6 {
			shrink = 0.6
		}
		fontPtsF *= shrink
	}

	fontPts := int(fontPtsF + 0.5)
	if fontPts < 6 {
		fontPts = 6
	}

	font := pickStandardFont(st.BaseFont)
	if opts.ForceBold || opts.ForceItalic {
		font = applyForcedWeight(font, opts.ForceBold, opts.ForceItalic)
	}
	r, g, b := 0.0, 0.0, 0.0
	if st.HaveCol {
		r, g, b = st.R, st.G, st.B
	}

	pageSel := []string{strconv.Itoa(m.Page)}

	// Step 1 — white-out using a generated white-PNG image stamp sized
	// to bbox + margin. Covers both vector text and raster image content
	// (relevant for scanned PDFs whose visible chars are pixels in an
	// image with an OCR text layer on top).
	//
	// Margins are asymmetric: OCR misalignment is predominantly
	// horizontal (kerning estimation), not vertical.
	//
	// In vector mode the pdftotext bbox is accurate to the glyph, so
	// horizontal padding has no upside and trims adjacent characters
	// (especially with italic fonts whose neighbours slant into the
	// margin). Vertical gets a tiny 1pt for descender safety.
	//
	// In scanned mode the OCR bbox is the source — typically 3-6pt off
	// horizontally — so we DO need horizontal padding.
	hMargin, vMargin := 0.0, 1.0
	if mode == ModeScanned {
		hMargin = 4.0
		vMargin = 2.5
	}
	coverW := int(bboxW + 2*hMargin + 0.5)
	coverH := int(bboxH + 2*vMargin + 0.5)
	bg := sampler.colorNear(m)
	coverFile, err := makeCover(coverW, coverH, bg)
	if err != nil {
		return fmt.Errorf("cover: %w", err)
	}
	defer os.Remove(coverFile)

	coverDesc := fmt.Sprintf(
		"scalefactor:1 abs, position:tl, offset:%.2f %.2f, opacity:1, rotation:0",
		m.X0-hMargin, -(m.Y0 - vMargin),
	)
	wm, err := api.ImageWatermark(coverFile, coverDesc, true, false, types.POINTS)
	if err != nil {
		return fmt.Errorf("whiteout wm: %w", err)
	}
	if err := api.AddWatermarksFile(file, "", pageSel, wm, model.NewDefaultConfiguration()); err != nil {
		return fmt.Errorf("apply whiteout: %w", err)
	}

	// Step 2 — stamp the new text on top with the recovered style.
	//
	// Baseline nudge: pdfcpu's text watermark anchors by the bbox of the
	// rendered glyphs (cap-height + descender), but pdftotext's bbox is
	// from glyph top to glyph bottom. The visual baseline of the original
	// text sits roughly 0.78 of the way down the bbox; ours lands a touch
	// lower because pdfcpu renders the bbox top at the offset Y. A small
	// upward shift compensates so the new text sits on the original
	// baseline.
	yNudge := float64(fontPts) * 0.10
	overlayDesc := fmt.Sprintf(
		"fontname:%s, points:%d, scalefactor:1 abs, position:tl, offset:%.2f %.2f, "+
			"fillcolor:%.3f %.3f %.3f, opacity:1, rotation:0, mode:0",
		font, fontPts, m.X0, -(m.Y0 - yNudge), r, g, b,
	)
	wm2, err := api.TextWatermark(newText, overlayDesc, true, false, types.POINTS)
	if err != nil {
		return fmt.Errorf("overlay wm: %w", err)
	}
	if err := api.AddWatermarksFile(file, "", pageSel, wm2, model.NewDefaultConfiguration()); err != nil {
		return fmt.Errorf("apply overlay: %w", err)
	}
	return nil
}

// pickStandardFont maps an arbitrary BaseFont name to one of pdfcpu's 14
// built-in standard fonts. We look at family hints (Times / Courier /
// Helvetica) and weight hints (Bold) and slant hints (Italic / Oblique).
//
// If we have no idea, fall back to Helvetica — it's the safest sans-serif
// choice and matches the most-used embedded fonts in modern PDFs.
func pickStandardFont(baseFont string) string {
	if baseFont == "" {
		return "Helvetica"
	}
	low := strings.ToLower(baseFont)

	bold := strings.Contains(low, "bold") || strings.Contains(low, "black") ||
		strings.Contains(low, "heavy")
	italic := strings.Contains(low, "italic") || strings.Contains(low, "oblique")

	switch {
	case strings.Contains(low, "courier") || strings.Contains(low, "mono"):
		switch {
		case bold && italic:
			return "Courier-BoldOblique"
		case bold:
			return "Courier-Bold"
		case italic:
			return "Courier-Oblique"
		default:
			return "Courier"
		}
	case strings.Contains(low, "times") || strings.Contains(low, "serif") ||
		strings.Contains(low, "georgia") || strings.Contains(low, "roman"):
		switch {
		case bold && italic:
			return "Times-BoldItalic"
		case bold:
			return "Times-Bold"
		case italic:
			return "Times-Italic"
		default:
			return "Times-Roman"
		}
	default:
		// sans-serif default (Helvetica / Arial / Verdana / Calibri / etc)
		switch {
		case bold && italic:
			return "Helvetica-BoldOblique"
		case bold:
			return "Helvetica-Bold"
		case italic:
			return "Helvetica-Oblique"
		default:
			return "Helvetica"
		}
	}
}

// applyForcedWeight rewrites the font name to the bold/italic variant
// of the same family when the user has explicitly toggled bold or
// italic. Falls back to the closest match in pdfcpu's standard 14 if the
// exact combination isn't available.
func applyForcedWeight(font string, bold, italic bool) string {
	family := "Helvetica"
	switch {
	case strings.HasPrefix(font, "Times"):
		family = "Times"
	case strings.HasPrefix(font, "Courier"):
		family = "Courier"
	}
	if family == "Times" {
		switch {
		case bold && italic:
			return "Times-BoldItalic"
		case bold:
			return "Times-Bold"
		case italic:
			return "Times-Italic"
		default:
			return "Times-Roman"
		}
	}
	suffix := ""
	if bold {
		suffix += "Bold"
	}
	if italic {
		if family == "Helvetica" {
			suffix += "Oblique"
		} else { // Courier
			suffix += "Oblique"
		}
	}
	if suffix == "" {
		return family
	}
	return family + "-" + suffix
}

func pad(n int) string {
	if n < 1 {
		n = 1
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
