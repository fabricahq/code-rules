// Verify tag listing parsing, annotated-tag records, and listing limits without Git.

package rules_test

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestParseTagAdvertisementKeepsTagAndPeeledObjects retains both records of an annotated tag.
func TestParseTagAdvertisementKeepsTagAndPeeledObjects(t *testing.T) {
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	got, err := rules.ParseTagAdvertisement(b + "\trefs/tags/release/1\n" + a + "\trefs/tags/release/1^{}\n" + a + "\trefs/tags/v1.0.0\n")
	want := map[string]string{"release/1": b, "release/1^{}": a, "v1.0.0": a}
	if err != nil || !maps.Equal(got, want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
	if got, err := rules.ParseTagAdvertisement(""); err != nil || len(got) != 0 {
		t.Fatalf("empty listing: %v, %v", got, err)
	}
}

// TestParseTagAdvertisementRejectsMalformedRecords refuses records Git would never produce for tags.
func TestParseTagAdvertisementRejectsMalformedRecords(t *testing.T) {
	a := strings.Repeat("a", 40)
	for name, line := range map[string]string{
		"uppercase object": strings.Repeat("A", 40) + "\trefs/tags/v1",
		"short object":     a[:39] + "\trefs/tags/v1",
		"missing tab":      a + " refs/tags/v1",
		"branch":           a + "\trefs/heads/main",
		"empty tag":        a + "\trefs/tags/",
		"carriage return":  a + "\trefs/tags/v1\r",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := rules.ParseTagAdvertisement(line + "\n")
			var refused *rules.TagAdvertisementError
			if got != nil || !errors.As(err, &refused) || refused.Kind != rules.InvalidTagAdvertisement {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}

// TestParseTagAdvertisementLimits accepts exactly 20,000 records and 8 MiB, and refuses more.
func TestParseTagAdvertisementLimits(t *testing.T) {
	var text strings.Builder
	for i := range 20_000 {
		fmt.Fprintf(&text, "%s\trefs/tags/ordinary-%d\n", strings.Repeat("a", 40), i)
	}
	if got, err := rules.ParseTagAdvertisement(text.String()); err != nil || len(got) != 20_000 {
		t.Fatalf("exact record limit: %d, %v", len(got), err)
	}
	text.WriteString(strings.Repeat("a", 40) + "\trefs/tags/another\n")
	var refused *rules.TagAdvertisementError
	if _, err := rules.ParseTagAdvertisement(text.String()); !errors.As(err, &refused) || refused.Kind != rules.TagLimitExceeded {
		t.Fatalf("over record limit: %v", err)
	}
	repeated := strings.Repeat(strings.Repeat("a", 40)+"\trefs/tags/v1\n", 20_000)
	if got, err := rules.ParseTagAdvertisement(repeated); err != nil || len(got) != 1 {
		t.Fatalf("exact record limit with one repeated tag: %d, %v", len(got), err)
	}
	if _, err := rules.ParseTagAdvertisement(repeated + strings.Repeat("a", 40) + "\trefs/tags/v1^{}\n"); !errors.As(err, &refused) || refused.Kind != rules.TagLimitExceeded {
		t.Fatalf("over record limit with repeated and peeled records: %v", err)
	}
	if _, err := rules.ParseTagAdvertisement(strings.Repeat("\n", 8*1024*1024)); err != nil {
		t.Fatalf("exact byte limit: %v", err)
	}
	if _, err := rules.ParseTagAdvertisement(strings.Repeat("\n", 8*1024*1024+1)); !errors.As(err, &refused) || refused.Kind != rules.TagLimitExceeded {
		t.Fatalf("over byte limit: %v", err)
	}
}
