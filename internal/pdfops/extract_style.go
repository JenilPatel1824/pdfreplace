package pdfops

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Style is what we manage to recover from the source PDF for a piece of text:
// the font name, point size, and fill color. Anything we couldn't determine
// stays at the zero-value and the caller falls back to a default.
type Style struct {
	FontKey  string  // resource key, e.g. "F1" — used to look up BaseFont
	BaseFont string  // PDF /BaseFont, e.g. "Helvetica-Bold" or "ABCDEF+Arial"
	Size     float64 // points (0 = unknown)
	R, G, B  float64 // fill color, 0..1 (defaults: 0,0,0 = black)
	HaveSize bool
	HaveCol  bool
}

// ExtractStyles opens the PDF, walks each page's content stream, and returns
// the Style that was active when the first occurrence of `needle` was rendered
// on that page. Pages where the needle isn't found are absent from the map.
//
// This is a best-effort scanner — it tracks the operators relevant to text
// styling (Tf, rg, g, k) and matches against text shown by Tj/TJ/'/" ops.
// PDFs with complex content streams (nested forms, encoded strings beyond
// basic ASCII, or unusual color spaces) will fall back to defaults.
func ExtractStyles(file string, needle string) (map[int]Style, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}

	out := make(map[int]Style)
	for p := 1; p <= ctx.PageCount; p++ {
		st, err := scanPageStyle(ctx, p, needle)
		if err != nil || st == nil {
			continue
		}
		out[p] = *st
	}
	return out, nil
}

func scanPageStyle(ctx *model.Context, page int, needle string) (*Style, error) {
	pageDict, _, _, err := ctx.PageDict(page, false)
	if err != nil {
		return nil, err
	}
	rdr, err := pdf.ExtractPageContent(ctx, page)
	if err != nil || rdr == nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rdr); err != nil {
		return nil, err
	}
	cs := buf.String()

	keyToBase := pageFontMap(ctx, pageDict)

	st := scanCS(cs, needle)
	if st == nil {
		return nil, nil
	}
	if base, ok := keyToBase[st.FontKey]; ok {
		st.BaseFont = base
	}
	return st, nil
}

// pageFontMap walks page.Resources.Font and returns a map of resource-key
// (e.g. "F1") to BaseFont string (e.g. "Helvetica-Bold"). Subset prefixes
// like "ABCDEF+" are stripped.
func pageFontMap(ctx *model.Context, pageDict types.Dict) map[string]string {
	out := map[string]string{}
	res, _ := ctx.DereferenceDict(pageDict["Resources"])
	if res == nil {
		return out
	}
	fonts, _ := ctx.DereferenceDict(res["Font"])
	if fonts == nil {
		return out
	}
	for k, v := range fonts {
		fd, _ := ctx.DereferenceDict(v)
		if fd == nil {
			continue
		}
		if bf, ok := fd["BaseFont"]; ok {
			name := nameValue(bf)
			if i := strings.Index(name, "+"); i >= 0 && i < 8 {
				name = name[i+1:]
			}
			out[k] = name
		}
	}
	return out
}

func nameValue(v types.Object) string {
	switch n := v.(type) {
	case types.Name:
		return n.Value()
	case *types.Name:
		return n.Value()
	default:
		return strings.TrimPrefix(fmt.Sprintf("%v", v), "/")
	}
}

// ---- content stream scanner -------------------------------------------------

// scanCS walks the content stream once. It maintains a small state machine:
//   tokens are pushed onto an operand stack; when we hit an operator we
//   consume operands and update state. We track:
//     * font key + raw font size (Tf)
//     * fill color (rg / g / k)
//     * text matrix (Tm) and the CTM stack (cm / q / Q) — only the
//       diagonal scale components, since that's what affects rendered size
//
// The EFFECTIVE font size we report is `raw_size * abs(text_matrix.d)
//   * abs(ctm.d)`.  Many real PDFs use `Tf font 1` + a Tm matrix that
// scales by the actual point size; missing this would give us size=1.
func scanCS(cs, needle string) *Style {
	needleLow := strings.ToLower(strings.Join(strings.Fields(needle), " "))

	tok := newTokenizer(cs)
	var ops []string

	cur := Style{R: 0, G: 0, B: 0}
	var rawSize float64
	tmA, tmD := 1.0, 1.0 // text matrix x/y scale (default identity)
	type ctm struct{ a, d float64 }
	ctmStack := []ctm{{1, 1}}
	curCTM := func() ctm { return ctmStack[len(ctmStack)-1] }
	inText := false

	commitSize := func() {
		c := curCTM()
		eff := rawSize * absF(tmD) * absF(c.d)
		cur.Size = eff
		cur.HaveSize = eff > 0
	}

	var found *Style
	flushCheck := func(text string) {
		if found != nil {
			return
		}
		if matchesNeedle(text, needleLow) {
			s := cur
			found = &s
		}
	}

	for {
		t, ok := tok.next()
		if !ok {
			break
		}
		if !t.isOp {
			ops = append(ops, t.val)
			continue
		}
		op := t.val
		switch op {
		case "q":
			ctmStack = append(ctmStack, curCTM())
		case "Q":
			if len(ctmStack) > 1 {
				ctmStack = ctmStack[:len(ctmStack)-1]
			}
			commitSize()
		case "cm":
			// a b c d e f cm — multiply CTM by this matrix
			if len(ops) >= 6 {
				a, _ := strconv.ParseFloat(ops[len(ops)-6], 64)
				d, _ := strconv.ParseFloat(ops[len(ops)-3], 64)
				top := &ctmStack[len(ctmStack)-1]
				top.a *= a
				top.d *= d
				commitSize()
			}
		case "BT":
			inText = true
			tmA, tmD = 1, 1
		case "ET":
			inText = false
		case "Tm":
			// a b c d e f Tm — replace text matrix
			if len(ops) >= 6 {
				a, _ := strconv.ParseFloat(ops[len(ops)-6], 64)
				d, _ := strconv.ParseFloat(ops[len(ops)-3], 64)
				tmA, tmD = a, d
				commitSize()
			}
		case "Tf":
			// /FontKey size Tf
			if len(ops) >= 2 {
				size, _ := strconv.ParseFloat(ops[len(ops)-1], 64)
				key := strings.TrimPrefix(ops[len(ops)-2], "/")
				cur.FontKey = key
				rawSize = size
				commitSize()
			}
		case "rg":
			if len(ops) >= 3 {
				cur.R, _ = strconv.ParseFloat(ops[len(ops)-3], 64)
				cur.G, _ = strconv.ParseFloat(ops[len(ops)-2], 64)
				cur.B, _ = strconv.ParseFloat(ops[len(ops)-1], 64)
				cur.HaveCol = true
			}
		case "g":
			if len(ops) >= 1 {
				v, _ := strconv.ParseFloat(ops[len(ops)-1], 64)
				cur.R, cur.G, cur.B = v, v, v
				cur.HaveCol = true
			}
		case "k":
			if len(ops) >= 4 {
				c, _ := strconv.ParseFloat(ops[len(ops)-4], 64)
				m, _ := strconv.ParseFloat(ops[len(ops)-3], 64)
				y, _ := strconv.ParseFloat(ops[len(ops)-2], 64)
				k, _ := strconv.ParseFloat(ops[len(ops)-1], 64)
				cur.R = (1 - c) * (1 - k)
				cur.G = (1 - m) * (1 - k)
				cur.B = (1 - y) * (1 - k)
				cur.HaveCol = true
			}
		case "Tj", "'":
			if inText && len(ops) >= 1 {
				flushCheck(decodePDFString(ops[len(ops)-1]))
			}
		case `"`:
			if inText && len(ops) >= 1 {
				flushCheck(decodePDFString(ops[len(ops)-1]))
			}
		case "TJ":
			if inText && len(ops) >= 1 {
				flushCheck(decodeTJArray(ops[len(ops)-1]))
			}
		}
		ops = ops[:0]
		if found != nil {
			break
		}
	}
	_ = tmA // reserved for future horizontal-scale checks
	return found
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func matchesNeedle(haystack, needleLow string) bool {
	hLow := strings.ToLower(strings.Join(strings.Fields(haystack), " "))
	return strings.Contains(hLow, needleLow)
}

// decodePDFString unwraps "(...)" and decodes basic escape sequences.
// Hex strings <...> are decoded too. Anything we can't decode we return as-is.
func decodePDFString(s string) string {
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		return decodeLiteral(s[1 : len(s)-1])
	}
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		return decodeHex(s[1 : len(s)-1])
	}
	return s
}

func decodeLiteral(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			break
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case '(', ')', '\\':
			b.WriteByte(s[i])
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func decodeHex(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
	if len(s)%2 == 1 {
		s += "0"
	}
	var b strings.Builder
	for i := 0; i < len(s); i += 2 {
		v, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			continue
		}
		b.WriteByte(byte(v))
	}
	return b.String()
}

// decodeTJArray pulls all string operands out of a TJ array literal
// like "[(He) -50 (llo)]" → "Hello".
var tjStr = regexp.MustCompile(`\(([^()\\]|\\.)*\)|<[0-9A-Fa-f\s]*>`)

func decodeTJArray(s string) string {
	var b strings.Builder
	for _, m := range tjStr.FindAllString(s, -1) {
		b.WriteString(decodePDFString(m))
	}
	return b.String()
}

// ---- tokenizer --------------------------------------------------------------

type token struct {
	val  string
	isOp bool
}

type tokenizer struct {
	s string
	i int
}

func newTokenizer(s string) *tokenizer { return &tokenizer{s: s} }

func (t *tokenizer) next() (token, bool) {
	for t.i < len(t.s) {
		c := t.s[t.i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == 0:
			t.i++
			continue
		case c == '%':
			// comment to end of line
			for t.i < len(t.s) && t.s[t.i] != '\n' && t.s[t.i] != '\r' {
				t.i++
			}
		case c == '(':
			return token{val: t.readLiteralString(), isOp: false}, true
		case c == '<':
			if t.i+1 < len(t.s) && t.s[t.i+1] == '<' {
				return token{val: t.readDict(), isOp: false}, true
			}
			return token{val: t.readHexString(), isOp: false}, true
		case c == '[':
			return token{val: t.readArray(), isOp: false}, true
		case c == '/':
			return token{val: t.readName(), isOp: false}, true
		case c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9'):
			return token{val: t.readNumber(), isOp: false}, true
		default:
			return token{val: t.readWord(), isOp: true}, true
		}
	}
	return token{}, false
}

func (t *tokenizer) readLiteralString() string {
	start := t.i
	depth := 0
	for t.i < len(t.s) {
		c := t.s[t.i]
		if c == '\\' && t.i+1 < len(t.s) {
			t.i += 2
			continue
		}
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				t.i++
				return t.s[start:t.i]
			}
		}
		t.i++
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readHexString() string {
	start := t.i
	for t.i < len(t.s) && t.s[t.i] != '>' {
		t.i++
	}
	if t.i < len(t.s) {
		t.i++
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readDict() string {
	start := t.i
	depth := 0
	for t.i+1 < len(t.s) {
		if t.s[t.i] == '<' && t.s[t.i+1] == '<' {
			depth++
			t.i += 2
			continue
		}
		if t.s[t.i] == '>' && t.s[t.i+1] == '>' {
			depth--
			t.i += 2
			if depth == 0 {
				return t.s[start:t.i]
			}
			continue
		}
		t.i++
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readArray() string {
	start := t.i
	depth := 0
	for t.i < len(t.s) {
		switch t.s[t.i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				t.i++
				return t.s[start:t.i]
			}
		case '(':
			t.readLiteralString()
			continue
		}
		t.i++
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readName() string {
	start := t.i
	t.i++ // /
	for t.i < len(t.s) {
		c := t.s[t.i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
			c == '/' || c == '[' || c == ']' || c == '(' || c == ')' ||
			c == '<' || c == '>' {
			break
		}
		t.i++
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readNumber() string {
	start := t.i
	if t.s[t.i] == '+' || t.s[t.i] == '-' {
		t.i++
	}
	for t.i < len(t.s) {
		c := t.s[t.i]
		if (c >= '0' && c <= '9') || c == '.' {
			t.i++
			continue
		}
		break
	}
	return t.s[start:t.i]
}

func (t *tokenizer) readWord() string {
	start := t.i
	for t.i < len(t.s) {
		c := t.s[t.i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '/' ||
			c == '[' || c == ']' || c == '(' || c == ')' ||
			c == '<' || c == '>' {
			break
		}
		t.i++
	}
	return t.s[start:t.i]
}
