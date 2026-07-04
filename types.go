package mailfold

// SendRequest is the payload for Client.Send. From is not included: the
// server always forces the From address to the mailbox the API key is
// bound to.
type SendRequest struct {
	To      []string `json:"to,omitempty"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
}

// SendResponse is returned by POST /api/v1/mail/send.
type SendResponse struct {
	Status string `json:"status"`
}

// Folder describes a single IMAP mailbox folder.
type Folder struct {
	Name       string   `json:"name"`
	Attributes []string `json:"attributes"`
}

// Address is a single email participant (name may be empty).
type Address struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// MessageHeader is a summary of a message, as returned by the listing and
// search endpoints.
type MessageHeader struct {
	UID     uint32    `json:"uid"`
	Subject string    `json:"subject"`
	From    []Address `json:"from"`
	To      []Address `json:"to"`
	Date    string    `json:"date"`
	Flags   []string  `json:"flags"`
	Seen    bool      `json:"seen"`
	Size    uint32    `json:"size"`
	Preview string    `json:"preview"`
}

// Attachment describes metadata about a message attachment (not its bytes;
// use Client.Attachment to fetch the bytes).
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

// Message is the full message body, as returned by GET .../mail/message.
type Message struct {
	MessageHeader
	Text        string       `json:"text"`
	HTML        string       `json:"html"`
	Attachments []Attachment `json:"attachments"`
}

// StatusResponse is the generic {"status": "..."} response body returned by
// the delete and flag endpoints.
type StatusResponse struct {
	Status string `json:"status"`
}

// Flag is one of the flag names accepted by Client.SetFlag.
type Flag string

const (
	FlagSeen     Flag = "seen"
	FlagFlagged  Flag = "flagged"
	FlagAnswered Flag = "answered"
	FlagDeleted  Flag = "deleted"
	FlagDraft    Flag = "draft"
)

// AttachmentData is the raw bytes of an attachment plus whatever metadata
// the server exposed via response headers.
//
// The attachment endpoint returns the file's raw bytes directly (not JSON),
// so this is a plain struct assembled from the HTTP response rather than an
// unmarshaled body.
type AttachmentData struct {
	Data        []byte
	ContentType string
	Filename    string
}
