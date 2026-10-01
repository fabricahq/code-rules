// Check exact ref classification and version-tag validation against shared expectations.

package rules_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/authored"
	"github.com/fabricahq/code-rules/internal/rules"
)

// projectedRef is a ref's canonical form as the fixtures describe it: a commit's lowercase SHA or a tag's full name.
type projectedRef struct {
	Kind rules.GitRefKind `json:"kind"`
	SHA  string           `json:"sha,omitempty"`
	Name string           `json:"name,omitempty"`
}

// projection returns ref's canonical form for comparison with a fixture.
func projection(ref rules.GitRef) projectedRef {
	if ref.Kind() == rules.GitRefCommit {
		return projectedRef{Kind: ref.Kind(), SHA: ref.Canonical()}
	}
	return projectedRef{Kind: ref.Kind(), Name: ref.Canonical()}
}

// TestRefsSharedExpectations checks values and typed errors through the public Go API.
func TestRefsSharedExpectations(t *testing.T) {
	data, err := os.ReadFile("testdata/refs/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID, Operation, Input, Location string
		Expected                       struct {
			OK    bool
			Value json.RawMessage
			Error *struct{ Name, Message, Location string }
		}
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		// Check each fixture independently so boundary failures identify their input.
		t.Run(test.ID, func(t *testing.T) {
			switch test.Operation {
			case "gitRef":
				got, err := rules.ParseGitRef(test.Input, test.Location)
				if !test.Expected.OK {
					var validation *authored.ValidationError
					if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
						t.Fatalf("got %v; want %s", err, test.Expected.Error.Message)
					}
					if !got.IsZero() {
						t.Fatalf("failure returned partial data: %#v", got)
					}
					return
				}
				var expected projectedRef
				if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
					t.Fatal(err)
				}
				if err != nil || projection(got) != expected || got.String() != test.Input {
					t.Fatalf("got %#v, %v; want %#v spelled %q", got, err, expected, test.Input)
				}
			case "tagVersion":
				got, err := rules.TagVersion(test.Input, test.Location)
				if !test.Expected.OK {
					var validation *authored.ValidationError
					if !errors.As(err, &validation) || err.Error() != test.Expected.Error.Message || validation.Location != test.Expected.Error.Location {
						t.Fatalf("got %v; want %s", err, test.Expected.Error.Message)
					}
					if got != "" {
						t.Fatalf("failure returned partial version: %q", got)
					}
					return
				}
				var expected string
				if err := json.Unmarshal(test.Expected.Value, &expected); err != nil {
					t.Fatal(err)
				}
				if err != nil || got != expected {
					t.Fatalf("got %q, %v; want %q", got, err, expected)
				}
			default:
				t.Fatalf("unhandled operation: %s", test.Operation)
			}
		})
	}
}

// ref parses text, failing the test when it isn't a valid ref.
func ref(t *testing.T, text string) rules.GitRef {
	t.Helper()
	parsed, err := rules.ParseGitRef(text, "ref")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// TestGitRef_ZeroValueMeansNoRef treats the zero value as no ref, empty in text and equal only to itself.
func TestGitRef_ZeroValueMeansNoRef(t *testing.T) {
	var none rules.GitRef
	if !none.IsZero() || none.String() != "" || none.Kind() != "" || none.Canonical() != "" || !none.Equal(rules.GitRef{}) {
		t.Fatalf("zero value %#v", none)
	}
	if tag := ref(t, "release/5"); tag.IsZero() || none.Equal(tag) || tag.Equal(none) {
		t.Fatal("a ref equals no ref")
	}
}

// TestGitRef_EqualComparesCanonicalFormsAndStringKeepsTheSpelling matches spellings of one revision while keeping
// each as authored, and tells kinds and revisions apart.
func TestGitRef_EqualComparesCanonicalFormsAndStringKeepsTheSpelling(t *testing.T) {
	sha := "ABCDEF0123456789ABCDEF0123456789ABCDEF01"
	for _, test := range []struct {
		a, b  string
		equal bool
	}{
		{"release/5", "release/5", true},
		{"release/5", "refs/tags/release/5", true},
		{sha, strings.ToLower(sha), true},
		{"release/5", "release/6", false},
		{"release/5", "refs/tags/release/50", false},
		{strings.ToLower(sha), "refs/tags/" + strings.ToLower(sha), false},
	} {
		a, b := ref(t, test.a), ref(t, test.b)
		if a.Equal(b) != test.equal || b.Equal(a) != test.equal {
			t.Errorf("%q and %q: equal %v, want %v", test.a, test.b, a.Equal(b), test.equal)
		}
		if a.String() != test.a || b.String() != test.b {
			t.Errorf("spellings %q and %q became %q and %q", test.a, test.b, a, b)
		}
	}
}

// TestGitRef_JSONCarriesTheAuthoredText encodes a ref as its authored text, omits a source's absent ref, and decodes
// the text back into an equal ref, refusing text that isn't a ref.
func TestGitRef_JSONCarriesTheAuthoredText(t *testing.T) {
	for _, text := range []string{"release/5", "refs/tags/release/5", "ABCDEF0123456789ABCDEF0123456789ABCDEF01"} {
		encoded, err := json.Marshal(rules.Source{Name: "team", Ref: ref(t, text)})
		if err != nil || !strings.Contains(string(encoded), `"ref":"`+text+`"`) {
			t.Fatalf("encoded %s, %v", encoded, err)
		}
		var decoded rules.Source
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Ref.String() != text || !decoded.Ref.Equal(ref(t, text)) {
			t.Fatalf("decoded %#v, %v", decoded.Ref, err)
		}
	}
	if encoded, err := json.Marshal(rules.Source{Name: "team"}); err != nil || strings.Contains(string(encoded), `"ref"`) {
		t.Fatalf("an absent ref was encoded: %s, %v", encoded, err)
	}
	var decoded rules.GitRef
	var validation *authored.ValidationError
	if err := json.Unmarshal([]byte(`"refs/heads/main"`), &decoded); !errors.As(err, &validation) || !decoded.IsZero() {
		t.Fatalf("decoded a branch: %#v, %v", decoded, err)
	}
}

// TestGitRefErrorLocation checks that the caller owns the diagnostic path.
func TestGitRefErrorLocation(t *testing.T) {
	_, err := rules.ParseGitRef("refs/heads/main", "config.sources.other.ref")
	var validation *authored.ValidationError
	if !errors.As(err, &validation) || validation.Location != "config.sources.other.ref" {
		t.Fatalf("unexpected error location: %v", err)
	}
}
