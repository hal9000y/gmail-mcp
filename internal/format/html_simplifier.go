package format

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// UnwrapTableLayout removes unnecessary single-column layout tables from HTML content.
// It recursively unwraps tables that are used purely for layout purposes while preserving
// semantic tables that contain actual data.
func UnwrapTableLayout(htmlContent []byte) []byte {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	maxIterations := 10
	for range maxIterations {
		changed := simplifyNode(doc)
		if !changed {
			break
		}
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlContent
	}

	return buf.Bytes()
}

func simplifyNode(n *html.Node) bool {
	changed := false

	// Process children first (bottom-up approach)
	child := n.FirstChild
	for child != nil {
		next := child.NextSibling
		if simplifyNode(child) {
			changed = true
		}
		child = next
	}

	// Then check if this node is a table that should be unwrapped
	if n.Type == html.ElementNode && n.Data == "table" {
		if shouldUnwrapTable(n) {
			unwrapTable(n)
			changed = true
		}
	}

	return changed
}

func shouldUnwrapTable(table *html.Node) bool {
	// Check if table has meaningful content (headers, multiple columns)
	if hasTableHeaders(table) {
		return false
	}

	columnCount := countTableColumns(table)
	if columnCount > 1 {
		return false
	}

	// For single-column tables, check if it has an ID that suggests it's structural
	// (like "main" in our email example) - these are usually layout tables
	for _, attr := range table.Attr {
		if attr.Key == "id" && (attr.Val == "main" || strings.Contains(attr.Val, "layout") || strings.Contains(attr.Val, "wrapper")) {
			return true
		}
	}

	// Check if it's a data table with multiple rows of actual content
	// and consistent structure (suggesting tabular data)
	rowCount := countContentRows(table)
	if rowCount > 5 && hasConsistentRowStructure(table) {
		return false
	}

	// Single column with few rows = likely a layout table
	return true
}

func hasTableHeaders(table *html.Node) bool {
	var hasHeaders bool
	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "th" || n.Data == "thead") {
			hasHeaders = true
			return
		}
		for c := n.FirstChild; c != nil && !hasHeaders; c = c.NextSibling {
			checkNode(c)
		}
	}
	checkNode(table)
	return hasHeaders
}

func countTableColumns(table *html.Node) int {
	maxCols := 0
	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			cols := 0
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cols++
				}
			}
			if cols > maxCols {
				maxCols = cols
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkNode(c)
		}
	}
	checkNode(table)
	return maxCols
}

func countContentRows(table *html.Node) int {
	rows := 0
	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			// Check if row has actual content (not just whitespace)
			if hasTextContent(n) {
				rows++
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkNode(c)
		}
	}
	checkNode(table)
	return rows
}

func hasTextContent(n *html.Node) bool {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		return text != "" && text != "&nbsp;"
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hasTextContent(c) {
			return true
		}
	}
	return false
}

func hasConsistentRowStructure(table *html.Node) bool {
	cellCounts := collectRowCellCounts(table)
	return areAllCountsEqual(cellCounts)
}

func collectRowCellCounts(table *html.Node) []int {
	var cellCounts []int
	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			cellCounts = append(cellCounts, countCellsInRow(n))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkNode(c)
		}
	}
	checkNode(table)
	return cellCounts
}

func countCellsInRow(row *html.Node) int {
	cols := 0
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cols++
		}
	}
	return cols
}

func areAllCountsEqual(counts []int) bool {
	if len(counts) < 2 {
		return false
	}
	firstCount := counts[0]
	for _, count := range counts[1:] {
		if count != firstCount {
			return false
		}
	}
	return true
}

func unwrapTable(table *html.Node) {
	// Extract content from table
	var content []*html.Node
	extractTableContent(table, &content)

	// Replace table with its content
	parent := table.Parent
	if parent != nil {
		for _, node := range content {
			parent.InsertBefore(node, table)
		}
		parent.RemoveChild(table)
	}
}

func extractTableContent(n *html.Node, content *[]*html.Node) {
	if n.Type == html.ElementNode && isTableElement(n.Data) {
		// Special handling for tr elements - add line break after each row
		if n.Data == "tr" {
			initialLen := len(*content)
			// Process the row's content
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extractTableContent(c, content)
			}
			// Add a line break after the row if it added any content
			if len(*content) > initialLen {
				lineBreak := &html.Node{
					Type: html.TextNode,
					Data: "\n",
				}
				*content = append(*content, lineBreak)
			}
		} else {
			// Skip other table wrapper elements, but process children
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				extractTableContent(c, content)
			}
		}
	} else if n.Type == html.ElementNode {
		// Keep other elements
		clone := cloneNode(n)
		*content = append(*content, clone)
	} else if n.Type == html.TextNode {
		if strings.TrimSpace(n.Data) != "" {
			clone := &html.Node{
				Type: html.TextNode,
				Data: n.Data,
			}
			*content = append(*content, clone)
		}
	}
}

func isTableElement(tag string) bool {
	return tag == "table" || tag == "tbody" || tag == "thead" ||
		tag == "tfoot" || tag == "tr" || tag == "td" || tag == "th"
}

func cloneNode(n *html.Node) *html.Node {
	clone := &html.Node{
		Type: n.Type,
		Data: n.Data,
		Attr: append([]html.Attribute{}, n.Attr...),
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		clone.AppendChild(cloneNode(c))
	}

	return clone
}
