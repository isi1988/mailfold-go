# mailfold-go

Official Go client SDK for [Mailfold](https://github.com/isi1988/Mailfold), a
self-hosted webmail/admin backend. This package wraps Mailfold's per-mailbox
REST API (send/read/search/delete mail, manage flags, fetch attachments)
using only the Go standard library — zero third-party runtime dependencies.

This is the **official client SDK** for the main project:
https://github.com/isi1988/Mailfold

## Why an API key instead of SMTP/IMAP?

You can always talk raw SMTP/IMAP to a mailbox — this API exists because it
removes work that protocol pair pushes onto every caller: one credential and
one HTTPS endpoint for both sending and reading (SMTP alone can't read),
never touching the real mailbox password (a leaked key is revoked on its
own, individually), built-in recipient/body-size caps and rate limiting, and
plain HTTPS on port 443 instead of mail ports (587/465/993) that plenty of
networks block outright. See the
[full rationale and capability list](https://github.com/isi1988/Mailfold#why-an-api-instead-of-talking-smtpimap-directly)
in the main project README.

**What this SDK can't do:** attach files when sending, move messages between
folders, create folders, touch calendars/contacts, or get real-time push for
new mail — those need a full webmail session, not an API key. See
["What an API key can and can't do"](https://github.com/isi1988/Mailfold#what-an-api-key-can-and-cant-do)
for the complete list.

## Install

```
go get github.com/isi1988/mailfold-go
```

## Quickstart

```go
package main

import (
	"errors"
	"fmt"
	"log"

	mailfold "github.com/isi1988/mailfold-go"
)

func main() {
	client := mailfold.New("https://your-mailfold-instance.example", "mf_live_...")

	// Send a message.
	if _, err := client.Send(mailfold.SendRequest{
		To:      []string{"friend@example.com"},
		Subject: "Hello from mailfold-go",
		Text:    "This was sent via the Mailfold API.",
	}); err != nil {
		log.Fatal(err)
	}

	// List folders.
	folders, err := client.Folders()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(folders)

	// List recent message headers.
	headers, err := client.Messages(mailfold.MessagesOptions{Folder: "INBOX", Limit: 10})
	if err != nil {
		log.Fatal(err)
	}

	uid := int(headers[0].UID)

	// Fetch a full message.
	msg, err := client.Message("INBOX", uid)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg.Subject, msg.Text)

	// Search a folder.
	results, err := client.Search("INBOX", "invoice")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(results), "matches")

	// Download an attachment's raw bytes.
	if len(msg.Attachments) > 0 {
		att, err := client.Attachment("INBOX", uid, 0)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(att.Filename, len(att.Data), "bytes")
	}

	// Flag a message as seen.
	if err := client.SetFlag(mailfold.SetFlagRequest{
		Folder: "INBOX",
		UID:    uid,
		Flag:   mailfold.FlagSeen,
		Set:    true,
	}); err != nil {
		log.Fatal(err)
	}

	// Delete a message.
	if err := client.DeleteMessage("INBOX", uid); err != nil {
		var apiErr *mailfold.APIError
		if errors.As(err, &apiErr) {
			fmt.Println("status:", apiErr.StatusCode, "message:", apiErr.Message)
			if apiErr.HasRetryAfter {
				fmt.Println("retry after:", apiErr.RetryAfter, "seconds")
			}
			return
		}
		log.Fatal(err)
	}
}
```

A complete runnable version of this example lives in [`example/main.go`](example/main.go).

## Authentication

Every request is authenticated with a per-mailbox API key sent as
`Authorization: Bearer <token>`. Tokens look like
`mf_live_<kid>_<secret>` — treat them as opaque strings; never parse them.

## Errors

Non-2xx responses are returned as `*mailfold.APIError`, which exposes:

- `StatusCode` — the HTTP status code
- `Message` — the server's `{"error": "..."}` message
- `RetryAfter` / `HasRetryAfter` — populated from the `Retry-After` header on
  `429` responses

## License

MIT — see [LICENSE](LICENSE).
