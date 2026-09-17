// Verify complete inventory bounds include unused assets and empty directories.

package library

import (
	"context"
	"errors"
	"testing"
)

// TestInventoryLimits checks exact bounds, aggregate bytes, entry counts, and cancellation.
func TestInventoryLimits(t *testing.T) {
	chunk := make([]byte, maxFileBytes)
	files := map[string][]byte{}
	for i := range 8 {
		files[string(rune('a'+i))] = chunk
	}
	if err := ValidateInventoryLimits(context.Background(), files, nil); err != nil {
		t.Fatal(err)
	}
	files["extra"] = []byte{1}
	if err := ValidateInventoryLimits(context.Background(), files, nil); err == nil {
		t.Fatal("aggregate limit ignored")
	}
	if err := ValidateInventoryLimits(context.Background(), map[string][]byte{"large": make([]byte, maxFileBytes+1)}, nil); err == nil {
		t.Fatal("file limit ignored")
	}
	if err := ValidateInventoryLimits(context.Background(), nil, make([]string, maxFiles)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInventoryLimits(context.Background(), map[string][]byte{"file": nil}, make([]string, maxFiles)); err == nil {
		t.Fatal("entry limit ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ValidateInventoryLimits(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
