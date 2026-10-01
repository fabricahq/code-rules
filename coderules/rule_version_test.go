// Check rule version ordering, advancement, and the largest-change rule for combined notes.

package coderules_test

import (
	"testing"

	"github.com/fabricahq/code-rules/coderules"
)

// TestRuleVersionNext_ResetsLowerComponents moves each change level to the documented next version.
func TestRuleVersionNext_ResetsLowerComponents(t *testing.T) {
	from := coderules.RuleVersion{Major: 1, Minor: 3, Patch: 2}
	for change, want := range map[coderules.Change]string{coderules.ChangeMajor: "2.0.0", coderules.ChangeMinor: "1.4.0", coderules.ChangePatch: "1.3.3", coderules.ChangeNew: "1.3.2", coderules.ChangeRetired: "1.3.2"} {
		got, err := from.Next(change)
		if err != nil || got.String() != want {
			t.Errorf("%s from %s: got %s, %v; want %s", change, from, got, err, want)
		}
	}
}

// TestRuleVersionCompare_OrdersNumericallyByComponent keeps 1.10.0 newer than 1.9.0.
func TestRuleVersionCompare_OrdersNumericallyByComponent(t *testing.T) {
	for _, test := range []struct {
		a, b string
		want int
	}{{"1.9.0", "1.10.0", -1}, {"2.0.0", "1.99.99", 1}, {"1.3.0", "1.3.0", 0}, {"1.3.1", "1.3.0", 1}} {
		a, errA := coderules.ParseRuleVersion(test.a, "a")
		b, errB := coderules.ParseRuleVersion(test.b, "b")
		if errA != nil || errB != nil {
			t.Fatal(errA, errB)
		}
		if got := a.Compare(b); got != test.want {
			t.Errorf("compare %s with %s: got %d, want %d", test.a, test.b, got, test.want)
		}
	}
}

// TestRuleVersionNext_RefusesToPassTheLargestNumber never produces a version the parser would reject.
func TestRuleVersionNext_RefusesToPassTheLargestNumber(t *testing.T) {
	for _, test := range []struct {
		from   string
		change coderules.Change
		want   string
	}{
		{"1.0.999999998", coderules.ChangePatch, "1.0.999999999"},
		{"1.0.999999999", coderules.ChangePatch, ""},
		{"1.999999999.4", coderules.ChangeMinor, ""},
		{"999999999.2.0", coderules.ChangeMajor, ""},
		{"1.999999999.4", coderules.ChangeMajor, "2.0.0"},
	} {
		from, err := coderules.ParseRuleVersion(test.from, "from")
		if err != nil {
			t.Fatal(err)
		}
		got, err := from.Next(test.change)
		if test.want == "" {
			if err == nil {
				t.Errorf("%s from %s: got %s, want an error", test.change, test.from, got)
			}
			continue
		}
		if err != nil || got.String() != test.want {
			t.Errorf("%s from %s: got %s, %v; want %s", test.change, test.from, got, err, test.want)
		}
		if _, err := coderules.ParseRuleVersion(got.String(), "next"); err != nil {
			t.Errorf("next version %s doesn't parse: %v", got, err)
		}
	}
}
