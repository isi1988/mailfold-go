package mailfold

import "fmt"

// APIError is returned whenever the Mailfold server responds with a
// non-2xx status code. It exposes the HTTP status, the server's error
// message, and (when the server set the header, typically on 429) the
// Retry-After value in seconds.
type APIError struct {
	StatusCode    int
	Message       string
	RetryAfter    int
	HasRetryAfter bool
}

func (e *APIError) Error() string {
	if e.HasRetryAfter {
		return fmt.Sprintf("mailfold: %d %s (retry after %ds)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("mailfold: %d %s", e.StatusCode, e.Message)
}
