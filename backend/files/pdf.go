package files

import (
	"bytes"
	"errors"
	"io"
	"os"

	pdf "rsc.io/pdf"
)

// ExtractPDFText opens a PDF at filePath and returns extracted text up to maxChars.
// It returns an error if the file can't be read. If maxChars <= 0, a sane default is used.
func ExtractPDFText(filePath string, maxChars int) (string, error) {
	if maxChars <= 0 {
		maxChars = 12000 // ~2-3k tokens, avoids blowing context
	}

	r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}

	// Some PDFs have no text layer; return empty string in that case
	var buf bytes.Buffer
	total := r.NumPage()
	for pageIndex := 1; pageIndex <= total; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		content := p.Content()
		for _, t := range content.Text {
			buf.WriteString(t.S)
		}
		buf.WriteString("\n\n")
		if buf.Len() >= maxChars {
			b := buf.Bytes()
			if len(b) > maxChars {
				b = b[:maxChars]
			}
			return string(b), nil
		}
	}

	// Fallback: If no text extracted, try raw content stream extraction
	if buf.Len() == 0 {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		if len(data) == 0 {
			return "", errors.New("pdf appears empty")
		}
		if len(data) > maxChars {
			data = data[:maxChars]
		}
		return string(bytes.ReplaceAll(data, []byte{'\x00'}, []byte{' '})), nil
	}
	// Trim at maxChars
	if buf.Len() > maxChars {
		return buf.String()[:maxChars], nil
	}
	return readAllString(&buf)
}

func readAllString(r io.Reader) (string, error) {
	var b bytes.Buffer
	if _, err := b.ReadFrom(r); err != nil {
		return "", err
	}
	return b.String(), nil
}
