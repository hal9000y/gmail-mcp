package format_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hal9000y/gmail-mcp/internal/format"
)

func TestPDF2Text(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not found in PATH")
	}

	override := os.Getenv("OVERRIDE") != ""
	cases := []struct {
		name     string
		pdfFile  string
		textFile string
	}{
		{
			name:     "general",
			pdfFile:  "./testdata/test.pdf",
			textFile: "./testdata/test.txt",
		},
	}

	cnv := format.Converter{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pdfData, err := os.ReadFile(tc.pdfFile)
			require.NoError(t, err, "failed to read PDF file")

			result, err := cnv.PDF2Text(pdfData)
			require.NoError(t, err, "PDF2Text failed")

			if override {
				err := os.WriteFile(tc.textFile, []byte(result), 0644)
				require.NoError(t, err, "failed to write override file")
				t.Log("Override mode: wrote output to", tc.textFile)
				return
			}

			expected, err := os.ReadFile(tc.textFile)
			require.NoError(t, err, "failed to read expected text file")

			assert.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(result), "PDF2Text output mismatch")
		})
	}
}

func TestHTML2MD(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("pandoc not found in PATH")
	}

	override := os.Getenv("OVERRIDE") != ""
	cases := []struct {
		name     string
		htmlFile string
		mdFile   string
	}{
		{
			name:     "email_with_layout_tables",
			htmlFile: "./testdata/email_layout.html",
			mdFile:   "./testdata/email_layout.md",
		},
		{
			name:     "semantic_content_with_data_tables",
			htmlFile: "./testdata/semantic_content.html",
			mdFile:   "./testdata/semantic_content.md",
		},
	}

	cnv := format.Converter{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			htmlData, err := os.ReadFile(tc.htmlFile)
			require.NoError(t, err, "failed to read HTML file")

			result, err := cnv.HTML2MD(htmlData)
			require.NoError(t, err, "HTML2MD failed")

			if override {
				err := os.WriteFile(tc.mdFile, []byte(result), 0644)
				require.NoError(t, err, "failed to write override file")
				t.Log("Override mode: wrote output to", tc.mdFile)
				return
			}

			expected, err := os.ReadFile(tc.mdFile)
			require.NoError(t, err, "failed to read expected MD file")

			assert.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(result), "HTML2MD output mismatch")
		})
	}
}
