# PDF Text Replacement — Architecture Notes

Real PDF text replacement is hard. The honest landscape of options for
this project:

## What this codebase does today (v1)

`internal/pdfops/replace.go` uses a **redact + overlay** approach:

1. `pdftotext -bbox-layout` finds every match of `oldText` and returns
   per-word bounding boxes.
2. We approximate the original font size from the box height
   (`fontPts ≈ height / 0.72`).
3. For each match we apply two pdfcpu watermarks: a white rectangle
   that covers the old text, then the new text in Helvetica on top.

### What works

- Layout stays stable — every other character on the page keeps its
  exact position.
- Replacement is fast (~50-200 ms per match).
- 100% pure-Go runtime — no MuPDF / Acrobat / cloud service needed.

### What doesn't

- **Original font isn't preserved.** Helvetica is used for everything.
  If the source uses a stylized font, the result will visibly differ.
- **Color isn't auto-detected.** Always black. Acceptable for the
  ~95% of PDFs that use black body text.
- **Overflow.** If `newText` is wider than `oldText`'s bounding box,
  it spills into adjacent content. The frontend warns the user.
- **Justified text doesn't re-flow.** The old characters' positions
  are gone — neighboring words don't move to compensate.

This is the right tradeoff for a *free* tool: 80% of users get
acceptable results, and we don't need a $5k UniPDF license or a Python
sidecar.

## Upgrade path A — MuPDF / mutool (best free)

`mutool` (part of MuPDF) supports proper PDF redaction and re-stamping
via the Page Operations API. Wrapping it via `go-fitz` gives:

- Original font preserved if it's embedded in the PDF.
- Original color preserved.
- Per-character text extraction (better than per-word).

Cost: cgo, ~30 MB binary, GPL/AGPL license unless you buy a commercial
license. Worth it once the tool has revenue.

To switch: implement `pdfops.Replace` against `go-fitz` — keep the
`Match` shape so `find.go` doesn't change.

## Upgrade path B — UniPDF (commercial)

[UniPDF](https://unidoc.io/unipdf/) by UniDoc gives you native Go text
extraction with font/color metadata and supports rewriting content
streams. It's the cleanest API by far.

Cost: $300/mo for the commercial license. Free for non-commercial use,
but the moment you run AdSense it becomes commercial.

## Upgrade path C — PyMuPDF sidecar

Run a tiny Python service alongside the Go server:

```python
# replacer.py — exposed at :9090/replace
import fitz, sys
doc = fitz.open(sys.argv[1])
for page in doc:
    for inst in page.search_for(sys.argv[2]):
        page.add_redact_annot(inst, sys.argv[3])
        page.apply_redactions()
doc.save(sys.argv[4])
```

Go calls it via `exec.Command("python", "replacer.py", ...)`. PyMuPDF
preserves font and color much better than our v1.

Cost: extra runtime dep, slightly more deploy complexity. Free
license-wise (AGPL — fine if you don't redistribute the binary).

## Recommendation by stage

| Stage                    | Use this        |
|--------------------------|-----------------|
| MVP / pre-revenue        | v1 (current)    |
| First $500/mo            | PyMuPDF sidecar |
| Scaling / pro features   | UniPDF or self-MuPDF wrapper |

## Future work for v1

- [ ] Detect text color from `pdftotext -bbox-layout`'s `<color>` info
      where available, and pass it as `fillc` to the overlay watermark.
- [ ] Auto-shrink overlay font when `newText` is wider than `oldText`
      (multiply `fontPts` by `oldWidth/newWidth` capped at 0.6).
- [ ] OCR fallback (`tesseract`) for scanned PDFs — gated behind a
      "this looks scanned" detection.
