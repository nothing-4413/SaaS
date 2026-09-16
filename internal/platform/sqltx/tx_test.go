package sqltx

import (
	"context"
	"testing"
)

func TestWithTxRejectsInvalidInput(t *testing.T) {
	if err := WithTx(context.Background(), nil, nil); err != ErrInvalidInput {
		t.Fatalf("err=%v", err)
	}
}
