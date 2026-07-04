// Command example demonstrates every method of the mailfold client.
// It is not run automatically; it exists so the SDK's usage compiles and
// stays in sync with the README's quickstart.
package main

import (
	"fmt"
	"log"

	mailfold "github.com/isi1988/mailfold-go"
)

func main() {
	client := mailfold.New("https://real.mailfold.site", "mf_live_0123456789abcdef_secret")

	if _, err := client.Send(mailfold.SendRequest{
		To:      []string{"friend@example.com"},
		Subject: "Hello from mailfold-go",
		Text:    "This was sent via the Mailfold API.",
	}); err != nil {
		log.Fatalf("send: %v", err)
	}

	folders, err := client.Folders()
	if err != nil {
		log.Fatalf("folders: %v", err)
	}
	fmt.Println("folders:", folders)

	headers, err := client.Messages(mailfold.MessagesOptions{Folder: "INBOX", Limit: 10})
	if err != nil {
		log.Fatalf("messages: %v", err)
	}
	if len(headers) == 0 {
		return
	}
	uid := int(headers[0].UID)

	msg, err := client.Message("INBOX", uid)
	if err != nil {
		log.Fatalf("message: %v", err)
	}
	fmt.Println("subject:", msg.Subject)

	results, err := client.Search("INBOX", "invoice")
	if err != nil {
		log.Fatalf("search: %v", err)
	}
	fmt.Println("search results:", len(results))

	if len(msg.Attachments) > 0 {
		att, err := client.Attachment("INBOX", uid, 0)
		if err != nil {
			log.Fatalf("attachment: %v", err)
		}
		fmt.Printf("attachment %q: %d bytes (%s)\n", att.Filename, len(att.Data), att.ContentType)
	}

	if err := client.SetFlag(mailfold.SetFlagRequest{
		Folder: "INBOX",
		UID:    uid,
		Flag:   mailfold.FlagSeen,
		Set:    true,
	}); err != nil {
		log.Fatalf("set flag: %v", err)
	}

	if err := client.DeleteMessage("INBOX", uid); err != nil {
		var apiErr *mailfold.APIError
		if ok := asAPIError(err, &apiErr); ok {
			fmt.Println("delete failed:", apiErr.StatusCode, apiErr.Message)
			return
		}
		log.Fatalf("delete: %v", err)
	}
}

func asAPIError(err error, target **mailfold.APIError) bool {
	if e, ok := err.(*mailfold.APIError); ok {
		*target = e
		return true
	}
	return false
}
