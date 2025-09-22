package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hal9000y/gmail-mcp/internal/format"
)

func TestUnwrapTableLayout(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single_column_layout_table",
			input: `<html><body>
				<table id="main">
					<tbody>
						<tr><td>Content line 1</td></tr>
						<tr><td>Content line 2</td></tr>
					</tbody>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				Content line 1
Content line 2

			</body></html>`,
		},
		{
			name: "nested_single_column_tables",
			input: `<html><body>
				<table>
					<tr><td>
						<table>
							<tr><td>
								<p>Nested content</p>
							</td></tr>
						</table>
					</td></tr>
				</table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t<p>Nested content</p>\n\n\t\t\t</body></html>",
		},
		{
			name: "data_table_with_headers",
			input: `<html><body>
				<table>
					<thead>
						<tr><th>Name</th><th>Age</th></tr>
					</thead>
					<tbody>
						<tr><td>Alice</td><td>30</td></tr>
						<tr><td>Bob</td><td>25</td></tr>
					</tbody>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				<table>
					<thead>
						<tr><th>Name</th><th>Age</th></tr>
					</thead>
					<tbody>
						<tr><td>Alice</td><td>30</td></tr>
						<tr><td>Bob</td><td>25</td></tr>
					</tbody>
				</table>
			</body></html>`,
		},
		{
			name: "multi_column_data_table",
			input: `<html><body>
				<table>
					<tbody>
						<tr><td>Cell 1</td><td>Cell 2</td><td>Cell 3</td></tr>
						<tr><td>Cell 4</td><td>Cell 5</td><td>Cell 6</td></tr>
					</tbody>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				<table>
					<tbody>
						<tr><td>Cell 1</td><td>Cell 2</td><td>Cell 3</td></tr>
						<tr><td>Cell 4</td><td>Cell 5</td><td>Cell 6</td></tr>
					</tbody>
				</table>
			</body></html>`,
		},
		{
			name: "table_with_layout_id",
			input: `<html><body>
				<table id="layout">
					<tr><td>Layout content</td></tr>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				Layout content

			</body></html>`,
		},
		{
			name: "table_with_wrapper_id",
			input: `<html><body>
				<table id="wrapper">
					<tr><td><div>Wrapped content</div></td></tr>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				<div>Wrapped content</div>

			</body></html>`,
		},
		{
			name: "consistent_multi_row_single_column",
			input: `<html><body>
				<table>
					<tbody>
						<tr><td>Row 1</td></tr>
						<tr><td>Row 2</td></tr>
						<tr><td>Row 3</td></tr>
						<tr><td>Row 4</td></tr>
						<tr><td>Row 5</td></tr>
						<tr><td>Row 6</td></tr>
						<tr><td>Row 7</td></tr>
					</tbody>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				<table>
					<tbody>
						<tr><td>Row 1</td></tr>
						<tr><td>Row 2</td></tr>
						<tr><td>Row 3</td></tr>
						<tr><td>Row 4</td></tr>
						<tr><td>Row 5</td></tr>
						<tr><td>Row 6</td></tr>
						<tr><td>Row 7</td></tr>
					</tbody>
				</table>
			</body></html>`,
		},
		{
			name: "empty_table",
			input: `<html><body>
				<table></table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t\n\t\t\t</body></html>",
		},
		{
			name: "table_with_only_whitespace",
			input: `<html><body>
				<table>
					<tr><td>   </td></tr>
					<tr><td>
					</td></tr>
				</table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t\n\t\t\t</body></html>",
		},
		{
			name: "mixed_content_preservation",
			input: `<html><body>
				<p>Before table</p>
				<table>
					<tr><td>
						<h1>Title</h1>
						<p>Paragraph</p>
						<ul>
							<li>Item 1</li>
							<li>Item 2</li>
						</ul>
					</td></tr>
				</table>
				<p>After table</p>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t<p>Before table</p>\n\t\t\t\t<h1>Title</h1><p>Paragraph</p><ul>\n\t\t\t\t\t\t\t<li>Item 1</li>\n\t\t\t\t\t\t\t<li>Item 2</li>\n\t\t\t\t\t\t</ul>\n\n\t\t\t\t<p>After table</p>\n\t\t\t</body></html>",
		},
		{
			name: "table_with_attributes",
			input: `<html><body>
				<table style="width:100%" class="layout">
					<tr><td align="center">
						<img src="logo.png" alt="Logo"/>
					</td></tr>
				</table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t<img src=\"logo.png\" alt=\"Logo\"/>\n\n\t\t\t</body></html>",
		},
		{
			name: "deeply_nested_tables",
			input: `<html><body>
				<table><tr><td>
					<table><tr><td>
						<table><tr><td>
							<table><tr><td>
								<p>Deep content</p>
							</td></tr></table>
						</td></tr></table>
					</td></tr></table>
				</td></tr></table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t<p>Deep content</p>\n\n\t\t\t</body></html>",
		},
		{
			name: "table_with_th_elements",
			input: `<html><body>
				<table>
					<tr><th>Header 1</th><th>Header 2</th></tr>
					<tr><td>Data 1</td><td>Data 2</td></tr>
				</table>
			</body></html>`,
			expected: "<html><head></head><body>\n\t\t\t\t<table>\n\t\t\t\t\t<tbody><tr><th>Header 1</th><th>Header 2</th></tr>\n\t\t\t\t\t<tr><td>Data 1</td><td>Data 2</td></tr>\n\t\t\t\t</tbody></table>\n\t\t\t</body></html>",
		},
		{
			name: "single_column_with_few_rows",
			input: `<html><body>
				<table>
					<tr><td>Row 1</td></tr>
					<tr><td>Row 2</td></tr>
					<tr><td>Row 3</td></tr>
				</table>
			</body></html>`,
			expected: `<html><head></head><body>
				Row 1
Row 2
Row 3

			</body></html>`,
		},
		{
			name:     "simple_paragraph",
			input:    `<html><body><p>Simple text</p></body></html>`,
			expected: `<html><head></head><body><p>Simple text</p></body></html>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := format.UnwrapTableLayout([]byte(tc.input))
			assert.Equal(t, tc.expected, string(result), "HTML output should match exactly")
		})
	}
}

func TestUnwrapTableLayoutWithRealEmail(t *testing.T) {
	// Test with actual email HTML from testdata
	emailHTML := `<html><body>
		<table id="main">
			<tbody>
				<tr><td>
					<table>
						<tbody>
							<tr><td>
								<img src="https://example.com/logo.png" alt="Company Logo"/>
							</td></tr>
							<tr><td>
								<p>Lorem ipsum dolor sit amet</p>
							</td></tr>
						</tbody>
					</table>
				</td></tr>
			</tbody>
		</table>
	</body></html>`

	expected := "<html><head></head><body>\n\t\t<img src=\"https://example.com/logo.png\" alt=\"Company Logo\"/><p>Lorem ipsum dolor sit amet</p>\n\n\t</body></html>"

	result := format.UnwrapTableLayout([]byte(emailHTML))
	assert.Equal(t, expected, string(result))
}

func TestUnwrapTableLayoutIdempotency(t *testing.T) {
	// Test that running the function multiple times produces the same result
	input := `<html><body>
		<table><tr><td>
			<p>Content</p>
		</td></tr></table>
	</body></html>`

	result1 := format.UnwrapTableLayout([]byte(input))
	result2 := format.UnwrapTableLayout(result1)
	result3 := format.UnwrapTableLayout(result2)

	assert.Equal(t, string(result1), string(result2), "Second run should produce same result")
	assert.Equal(t, string(result2), string(result3), "Third run should produce same result")
}

func TestUnwrapTableLayoutErrorHandling(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid_html",
			input: `<html><body><table><tr><td>Unclosed`,
		},
		{
			name:  "malformed_nesting",
			input: `<table><td><tr>Wrong nesting</tr></td></table>`,
		},
		{
			name:  "empty_input",
			input: ``,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_ = format.UnwrapTableLayout([]byte(tc.input))
			}, "Should not panic on %s", tc.name)
		})
	}
}

func TestUnwrapTableLayoutComplexNesting(t *testing.T) {
	// Test with complex real-world email structure
	input := `<html><body>
		<table id="wrapper" width="100%">
			<tr><td align="center">
				<table id="main" width="600">
					<tr><td>
						<table width="100%">
							<tr><td>Header Content</td></tr>
						</table>
					</td></tr>
					<tr><td>
						<table width="100%">
							<tr><td>Body Content</td></tr>
						</table>
					</td></tr>
					<tr><td>
						<table width="100%">
							<tr><td>Footer Content</td></tr>
						</table>
					</td></tr>
				</table>
			</td></tr>
		</table>
	</body></html>`

	expected := `<html><head></head><body>
		Header ContentBody ContentFooter Content

	</body></html>`

	result := format.UnwrapTableLayout([]byte(input))
	assert.Equal(t, expected, string(result))
}
