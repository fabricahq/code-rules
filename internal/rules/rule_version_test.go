// Check rule version ordering, advancement, and the largest-change rule for combined notes.

package rules_test

import (
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestRuleVersionNext_ResetsLowerComponents moves each change level to the documented next version.
func TestRuleVersionNext_ResetsLowerComponents(t *testing.T) {
	from := rules.RuleVersion{Major: 1, Minor: 3, Patch: 2}
	for change, want := range map[rules.Change]string{rules.ChangeMajor: "2.0.0", rules.ChangeMinor: "1.4.0", rules.ChangePatch: "1.3.3", rules.ChangeNew: "1.3.2", rules.ChangeRetired: "1.3.2"} {
		if got := from.Next(change).String(); got != want {
			t.Errorf("%s from %s: got %s, want %s", change, from, got, want)
		}
	}
}

// TestRuleVersionCompare_OrdersNumericallyByComponent keeps 1.10.0 newer than 1.9.0.
func TestRuleVersionCompare_OrdersNumericallyByComponent(t *testing.T) {
	for _, test := range []struct {
		a, b string
		want int
	}{{"1.9.0", "1.10.0", -1}, {"2.0.0", "1.99.99", 1}, {"1.3.0", "1.3.0", 0}, {"1.3.1", "1.3.0", 1}} {
		a, errA := rules.ParseRuleVersion(test.a, "a")
		b, errB := rules.ParseRuleVersion(test.b, "b")
		if errA != nil || errB != nil {
			t.Fatal(errA, errB)
		}
		if got := a.Compare(b); got != test.want {
			t.Errorf("compare %s with %s: got %d, want %d", test.a, test.b, got, test.want)
		}
	}
}

// TestLargerChange_PicksTheLargestVersionChange resolves several pending notes on one rule.
func TestLargerChange_PicksTheLargestVersionChange(t *testing.T) {
	for _, test := range []struct{ a, b, want rules.Change }{
		{rules.ChangePatch, rules.ChangeMajor, rules.ChangeMajor},
		{rules.ChangeMajor, rules.ChangeMinor, rules.ChangeMajor},
		{rules.ChangeMinor, rules.ChangePatch, rules.ChangeMinor},
		{rules.ChangePatch, rules.ChangePatch, rules.ChangePatch},
	} {
		if got := rules.LargerChange(test.a, test.b); got != test.want {
			t.Errorf("larger of %s and %s: got %s, want %s", test.a, test.b, got, test.want)
		}
	}
}
