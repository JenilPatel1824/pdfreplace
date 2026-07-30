package pdfops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)


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

func RenderPage(pdf, page string) ([]byte, error) {
	return RenderPageDPI(pdf, page, 110)
}


func RenderPageDPI(pdf, page string, dpi int) ([]byte, error) {
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
		"-png", "-r", strconv.Itoa(dpi),
		"-f", strconv.Itoa(n), "-l", strconv.Itoa(n),
		"-singlefile", pdf, prefix)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w (%s)", err, stderr.String())
	}
	return os.ReadFile(prefix + ".png")
}


func runPdftotextPage(pdf string, page int) (string, error) {
	if page < 1 {
		return "", fmt.Errorf("bad page %d", page)
	}
	cmd := exec.Command("pdftotext",
		"-f", strconv.Itoa(page), "-l", strconv.Itoa(page),
		pdf, "-")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w (%s)", err, errBuf.String())
	}
	return out.String(), nil
}
