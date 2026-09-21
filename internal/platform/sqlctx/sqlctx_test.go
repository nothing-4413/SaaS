package sqlctx

import (
	"testing"
	"time"
)

func TestContextHasBoundedDeadline(t *testing.T) {
	ctx := Context()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("context has no deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > 5*time.Second {
		t.Fatalf("unexpected deadline window: %s", remaining)
	}
}
