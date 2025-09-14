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
	cmdPdfToHTML = "pdftohtml"
)

// Converter handles document format conversions.
type Converter struct{}

// PDF2MD converts PDF content to Markdown.
func (c Converter) PDF2MD(raw []byte) (string, error) {
	tmpPDF, err := os.CreateTemp("", "pdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("os.CreateTemp failed: %w", err)
	}
	defer func() {
		if err := tmpPDF.Close(); err != nil {
			log.Println(fmt.Errorf("tmpPDF.Close failed: %w", err))
		}
		if err := os.Remove(tmpPDF.Name()); err != nil {
			log.Println(fmt.Errorf("os.Remove(%s) failed: %w", tmpPDF.Name(), err))
		}
	}()

	if _, err := tmpPDF.Write(raw); err != nil {
		return "", fmt.Errorf("tmpPDF.Write failed: %w", err)
	}

	// Convert PDF to HTML using pdftohtml
	// -s: generate single HTML page
	// -i: ignore images
	// -noframes: single file output  
	// -stdout: output to stdout
	cmd := exec.Command(cmdPdfToHTML, "-s", "-i", "-noframes", "-stdout", tmpPDF.Name())
	htmlOutput, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftohtml failed: %w", err)
	}

	// Convert HTML to Markdown
	return c.HTML2MD(htmlOutput)
}

// HTML2MD converts HTML content to Markdown.
func (c Converter) HTML2MD(raw []byte) (string, error) {
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

	if _, err := tmpHTML.Write(raw); err != nil {
		return "", fmt.Errorf("tmpHTML.Write failed: %w", err)
	}

	cmd := exec.Command(cmdPandoc, "-f", "html", "-t", "markdown", "--wrap=none", tmpHTML.Name())
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pandoc conversion failed: %w", err)
	}

	return string(output), nil
}
