// Package format provides document format conversion utilities.
package format

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	cmdPandoc    = "pandoc"
	cmdPdfToText = "pdftotext"
)

// Converter handles document format conversions.
type Converter struct{}

// HTML2MD converts HTML content to Markdown.
func (c Converter) HTML2MD(raw []byte) (string, error) {
	simplified := UnwrapTableLayout(raw)

	tmpHTML, err := os.CreateTemp("", "html-*.html")
	if err != nil {
		return "", fmt.Errorf("os.CreateTemp failed: %w", err)
	}
	defer func() {
		if err := tmpHTML.Close(); err != nil {
			log.Println(fmt.Errorf("tmpHTML.Close failed: %w", err))
		}
		if err := os.Remove(tmpHTML.Name()); err != nil {
			log.Println(fmt.Errorf("os.Remove(%s) failed: %w", tmpHTML.Name(), err))
		}
	}()

	if _, err := tmpHTML.Write(simplified); err != nil {
		return "", fmt.Errorf("tmpHTML.Write failed: %w", err)
	}

	// Use CommonMark format for optimal balance of structure preservation and token efficiency
	// CommonMark preserves links, emphasis, and lists while being ~50% smaller than original HTML
	cmd := exec.Command(cmdPandoc, "-f", "html", "-t", "commonmark", "--wrap=none", tmpHTML.Name())
	log.Printf("Running command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pandoc conversion failed: %w", err)
	}

	return string(output), nil
}

// PDF2Text extracts plain text from PDF content.
func (c Converter) PDF2Text(raw []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdfconv-*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp failed: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Println(fmt.Errorf("os.RemoveAll(%s) failed: %w", tmpDir, err))
		}
	}()

	pdfPath := tmpDir + "/document.pdf"
	if err := os.WriteFile(pdfPath, raw, 0600); err != nil {
		return "", fmt.Errorf("os.WriteFile failed: %w", err)
	}

	// Convert PDF to text using pdftotext
	// -layout: maintain original physical layout
	// -: output to stdout
	cmd := exec.Command(cmdPdfToText, "-layout", pdfPath, "-")
	log.Printf("Running command: %s", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}

	return string(output), nil
}
