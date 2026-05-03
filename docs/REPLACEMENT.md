# Text Replacement — Architecture & Limits

This doc explains how text replacement works in the codebase today, what
its limits are, and the upgrade paths for getting truly pixel-perfect
results.

## What v1 does (current)

`internal/pdfops/replace.go` runs a four-stage pipeline:

1. **Find.** `pdftotext -bbox-layout` extracts every glyph's bounding
   box. We walk those, build per-line strings, and substring-match the
   user's "old" text — yielding a list of `Match{page, x0,y0,x1,y1}`
   tuples.
2. **Style extract** (vector mode only). `internal/pdfops/extract_style.go`
   tokenises the page's PDF content stream and tracks the active font
   key, raw font size, text-matrix scale, CTM stack, and fill color
   when each text-show op fires. The base font name is resolved via
   the page's `/Resources /Font` dictionary.
3. **Cover.** `internal/pdfops/cover.go` renders the page once, samples
   pixels just outside the match's bbox (filtering out dark text/border
   pixels), and takes a per-channel median for the background color.
   We generate a tiny PNG sized to bbox + asymmetric margins (0pt H /
   1pt V in vector mode, 4pt / 2.5pt in scanned mode) filled with that
   color, and stamp it as a pdfcpu image watermark with `onTop=true`.
4. **Stamp.** A second pdfcpu text watermark with the new text — using
   the recovered font/size/color, mapped to one of pdfcpu's 14 standard
   fonts via `pickStandardFont`. A 10% upward Y nudge compensates for
   pdfcpu's bbox vs pdftotext's glyph-bbox conventions. If the new text
   would overflow the original bbox, the font is shrunk proportionally
   (capped at 0.6×).

### What works well today

- ✅ Helvetica / Arial / Verdana / Calibri / common sans-serif sources →
  visually identical replacement (Helvetica family used).
- ✅ Times / Georgia / Cambria / common serif sources → close-enough
  via Times-Roman family.
- ✅ Courier / Consolas / monospace → Courier family.
- ✅ Bold / Italic / Bold-Italic detection from BaseFont name.
- ✅ Manual Bold / Italic override pills in editor when auto-detect is
  wrong.
- ✅ Color preservation when the source uses `rg`/`g`/`k` (most PDFs).
- ✅ Background-matched cover (no white sticker on coloured paper).
- ✅ Position to within ~1pt of original baseline.
- ✅ Width-aware shrink prevents overflow into adjacent text.

### Known limits (v1)

| Source font category | Result | Why |
|---|---|---|
| Standard sans-serif/serif/mono | Pixel-equivalent | We have a matching standard font |
| Bold / Italic variants | Pixel-equivalent | Standard 14 covers all 4 combos |
| Decorative / script (e.g. wedding invite, certificate) | Helvetica fallback | Not in pdfcpu's standard 14 |
| CJK / Arabic / Hebrew | Likely fails | Standard 14 are Latin-1 only |
| Heavily kerned / ligatured display fonts | Visible difference | We can't reproduce kerning tables of an arbitrary font |

These are fundamental to the v1 architecture. The replacement glyph
shapes come from pdfcpu's bundled standard fonts, regardless of what
the source PDF uses.

## v2 — Embedded font reuse (research notes)

### The goal

Read the source PDF's embedded font program, use it for the
replacement text. Replacement glyphs become pixel-identical.

### What pdfcpu provides

```go
pdfcpu.ExtractPageFonts(ctx, pageNr) ([]Font, error)  // io.Reader of font binary
api.InstallFonts(filenames []string) error            // register a TTF/OTF
```

After install, you reference the font by file basename in any
text-watermark `fontname:` parameter.

### What's hard

1. **Subsetted fonts.** Most PDFs embed only the glyphs they actually
   use, with a 6-character random prefix on the BaseFont name
   (e.g. `ABCDEF+EdwardianScriptITC`). The TTF you extract contains
   only those glyphs — characters in `newText` that weren't in the
   original render as the `.notdef` box.

2. **Encoding remap.** Embedded fonts often carry a custom `/Encoding`
   or `/CIDToGIDMap` rerouting character codes to glyphs. pdfcpu's
   text rendering expects normal Unicode mapping. We'd need to either
   strip + rebuild a Unicode `cmap` table before installing, or
   re-emit raw text-show ops in the same encoding.

3. **Type 3, Type 0 (CID), Type 1 (PostScript)**. Several PDF font
   types aren't TTF/OTF — `api.InstallFonts` rejects them.

4. **Font licensing.** Subsetting fonts inside a single rendered
   document is permitted by virtually every font EULA. Re-embedding
   them in a *new* document we generate is usually fine but on shakier
   ground for some commercial fonts. Safe path: re-embed only into the
   modified document, never expose as downloadable, delete the temp
   font file at end-of-request.

### Pragmatic plan

**Phase A — ship a curated open-font set.** Bundle ~10 popular
open-source faces (Inter, Source Sans, Roboto, Source Serif, Lora,
Merriweather, JetBrains Mono, Playfair Display, Caveat, Pacifico),
register them at startup via `api.InstallFonts`, expand
`pickStandardFont` to map common BaseFonts to them.

This catches a huge slice of "modern web-export PDF" cases (everything
made via Google Docs, Notion, modern report tools) without any
extraction. Effort: 1 day.

**Phase B — runtime font extraction.**
- For each match, find the source's BaseFont via the existing extractor.
- Skip if BaseFont matches one we already bundle.
- Otherwise call `ExtractPageFonts(ctx, page)`, find the right one, save
  to temp file.
- Probe `cmap` coverage with `golang.org/x/image/font/sfnt` — make sure
  every char of `newText` has a glyph.
- If yes: `api.InstallFonts(...)` then use that font name in the
  overlay watermark.
- If no: fall back to v1 standard-font matching.
- Clean up the temp font file at request end.

Effort: 2-3 days. Benefits ~10-20% of replacements. Defer until ad
revenue justifies the overhead, OR a high-value use-case demands it.

## v2 alternative — MuPDF / PyMuPDF sidecar

`mutool` and PyMuPDF (`fitz`) implement proper content-stream-level
edits: they understand the original font, can re-emit `Tj` / `TJ` with
new text, and preserve everything else.

```python
# replace_one.py
import fitz, sys
doc = fitz.open(sys.argv[1])
needle, repl = sys.argv[2], sys.argv[3]
for page in doc:
    for inst in page.search_for(needle):
        page.add_redact_annot(inst, repl)
        page.apply_redactions()
doc.save(sys.argv[4])
```

Spawn from Go via `exec.Command`.

**Pros**: Best-in-class quality. Free (AGPL).
**Cons**: Adds Python to the deploy. AGPL is fine for hosted SaaS but
not for redistributing as a binary.

## v2 alternative — UniPDF

`github.com/unidoc/unipdf` (commercial) gives proper content-stream
editing in pure Go.

**Pros**: One library swap; no Python in deploy.
**Cons**: $300+/month for the commercial licence. The free trial puts
a watermark on every output, which defeats our "no watermark"
promise.

## Decision matrix

| Stage | Recommendation |
|---|---|
| MVP / pre-revenue (now) | v1. Document limits in editor's tips card. |
| 1k+ users / month | Phase A — bundle 10 open fonts. ~1 day work, big quality bump for branded-export PDFs. |
| $100/month in ads | Phase B — runtime extraction. Or jump straight to PyMuPDF sidecar. |
| $500+/month | PyMuPDF sidecar (free, AGPL) or UniPDF (commercial) — operations vs licence cost. |
