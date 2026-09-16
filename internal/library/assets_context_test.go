// Verify that dependency discovery observes cancellation even when no file read remains.

package library

import (
	"context"
	"errors"
	"testing"
)

// TestSupportingLinksCanceledWithoutReads covers link-only work after catalog files are already in memory.
func TestSupportingLinksCanceledWithoutReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, document := range []string{"plain text", "[external](https://example.com)", "[self]()"} {
		r := reader{ctx: ctx, files: map[string][]byte{"techs/go/r.md": []byte(document)}}
		if err := r.supportingLinks(nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("%q: %v", document, err)
		}
	}
}
