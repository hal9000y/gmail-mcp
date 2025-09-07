package format

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	cmdPandoc    = "pandoc"
	cmdPdfToHTML = "pdftohtml"
)

type Converter struct{}

func (c Converter) PDF2MD(raw []byte) (string, error) {
	tmpPDF, err := os.CreateTemp("", "pdf-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp PDF file: %w", err)
	}
	defer os.Remove(tmpPDF.Name())

	if _, err := tmpPDF.Write(raw); err != nil {
		tmpPDF.Close()
		return "", fmt.Errorf("failed to write PDF data: %w", err)
	}
	tmpPDF.Close()

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

func (c Converter) HTML2MD(raw []byte) (string, error) {
	tmpHTML, err := os.CreateTemp("", "html-*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	defer os.Remove(tmpHTML.Name())

	if _, err := tmpHTML.Write(raw); err != nil {
		tmpHTML.Close()
		return "", fmt.Errorf("failed to write HTML data: %w", err)
	}
	tmpHTML.Close()

	cmd := exec.Command(cmdPandoc, "-f", "html", "-t", "markdown", "--wrap=none", tmpHTML.Name())
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pandoc conversion failed: %w", err)
	}

	return string(output), nil
}
