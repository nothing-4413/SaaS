package sqlctx

import (
	"context"
	"time"
)

// Context bounds one database operation or transaction so a stalled database
// cannot hold an HTTP request or worker loop indefinitely.
func Context() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// Call cancel when the deadline is reached so the cancel function is not
	// discarded. Callers receive a short-lived context and do not need to
	// manage a second return value for every store operation.
	time.AfterFunc(5*time.Second, cancel)
	return ctx
}
