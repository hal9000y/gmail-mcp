package tool

// EmailAddress represents an email address with optional display name.
type EmailAddress struct {
	Name  string `json:"name,omitempty" jsonschema:"the display name"`
	Email string `json:"email" jsonschema:"the email address"`
}

// MessageSummary contains essential message metadata.
type MessageSummary struct {
	ID        string         `json:"id" jsonschema:"message ID"`
	ThreadID  string         `json:"thread_id" jsonschema:"thread ID"`
	Timestamp string         `json:"timestamp" jsonschema:"message timestamp"`
	From      EmailAddress   `json:"from" jsonschema:"sender information"`
	To        []EmailAddress `json:"to,omitempty" jsonschema:"recipients"`
	CC        []EmailAddress `json:"cc,omitempty" jsonschema:"CC recipients"`
	Subject   string         `json:"subject" jsonschema:"email subject"`
	Snippet   string         `json:"snippet" jsonschema:"message preview"`
}
