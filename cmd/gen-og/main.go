// gen-og writes share images, publisher logo, and touch icon into web/static/.
// Run from repo root: go run ./cmd/gen-og
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	dark := color.RGBA{R: 0x16, G: 0x18, B: 0x2e, A: 0xff}
	blue := color.RGBA{R: 0x1e, G: 0x3a, B: 0x5f, A: 0xff}
	must(writePNGRect("web/static/og-home.png", 1200, 630, dark))
	must(writePNGRect("web/static/og-editor.png", 1200, 630, blue))
	// Organization / Article publisher logo (crawlable PNG for structured data).
	must(writePNGRect("web/static/logo-publisher.png", 600, 60, dark))
	must(writePNGRect("web/static/apple-touch-icon.png", 180, 180, dark))
}

func writePNGRect(path string, w, h int, c color.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return png.Encode(f, img)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
