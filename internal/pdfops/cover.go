package pdfops

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sort"
	"strconv"
)

// makeCover writes a w×h PNG of solid color `c` to a temp file and
// returns the path. Caller is responsible for deleting it.
//
// Each PNG pixel renders as one PDF point under pdfcpu's default DPI
// scaling, so we can size the cover to a bbox in points exactly.
func makeCover(w, h int, c color.RGBA) (string, error) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	f, err := os.CreateTemp("", "pdfrep-cover-*.png")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// bgSampler renders a PDF page once and lets you sample the background
// color around any bbox on it. We cache the rendered image because a
// single replace can stamp dozens of matches per page.
type bgSampler struct {
	pdf   string
	cache map[int]image.Image
}

func newBGSampler(pdf string) *bgSampler {
	return &bgSampler{pdf: pdf, cache: map[int]image.Image{}}
}

func (s *bgSampler) page(p int) (image.Image, error) {
	if img, ok := s.cache[p]; ok {
		return img, nil
	}
	bb, err := RenderPage(s.pdf, strconv.Itoa(p))
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(bb))
	if err != nil {
		return nil, err
	}
	s.cache[p] = img
	return img, nil
}

// renderDPI must match what RenderPage uses, otherwise our sampling
// coordinates are wrong.
const renderDPI = 110.0

// colorNear returns the median background color sampled from pixels just
// outside the match's bbox. We sample short bands above, below, left,
// and right of the bbox and take the per-channel median, ignoring dark
// pixels (which would be other text or borders, not background).
//
// If the page can't be rendered or no usable samples are found, returns
// white as the safe default.
func (s *bgSampler) colorNear(m Match) color.RGBA {
	img, err := s.page(m.Page)
	if err != nil {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	scale := renderDPI / 72.0
	bnd := img.Bounds()
	px0 := int(m.X0 * scale)
	py0 := int(m.Y0 * scale)
	px1 := int(m.X1 * scale)
	py1 := int(m.Y1 * scale)

	var rs, gs, bs []uint8
	add := func(x, y int) {
		if x < bnd.Min.X || y < bnd.Min.Y || x >= bnd.Max.X || y >= bnd.Max.Y {
			return
		}
		r, g, b, _ := img.At(x, y).RGBA()
		R := uint8(r >> 8)
		G := uint8(g >> 8)
		B := uint8(b >> 8)
		// luminance ≈ 0.299R + 0.587G + 0.114B
		lum := (uint32(R)*299 + uint32(G)*587 + uint32(B)*114) / 1000
		if lum < 130 {
			// skip dark pixels — almost certainly text or borders, not bg
			return
		}
		rs = append(rs, R)
		gs = append(gs, G)
		bs = append(bs, B)
	}

	// vertical bands above and below (5 rows each, 4-pixel stride)
	for off := 6; off <= 14; off += 2 {
		for x := px0; x <= px1; x += 4 {
			add(x, py0-off)
			add(x, py1+off)
		}
	}
	// horizontal bands left and right
	for off := 6; off <= 14; off += 2 {
		for y := py0; y <= py1; y += 4 {
			add(px0-off, y)
			add(px1+off, y)
		}
	}

	if len(rs) == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.RGBA{
		R: medianU8(rs),
		G: medianU8(gs),
		B: medianU8(bs),
		A: 255,
	}
}

func medianU8(vals []uint8) uint8 {
	if len(vals) == 0 {
		return 255
	}
	cp := append([]uint8(nil), vals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[len(cp)/2]
}
