// Verify revision interpretation at the project input boundary without subprocesses.

package project

import "testing"

func TestSourceRefSelections(t *testing.T) {
	for _, tc := range []struct{ input, field, value string }{
		{"0123456789012345678901234567890123456789", "ref", "0123456789012345678901234567890123456789"},
		{"v1.2.3", "ref", "v1.2.3"},
		{"1.2.3", "ref", "1.2.3"},
		{"release/stable", "ref", "release/stable"},
		{"refs/tags/=special", "ref", "refs/tags/=special"},
		{">= 1.2.0, < 2.0.0", "version", ">= 1.2.0, < 2.0.0"},
		{"~> 1.2.0", "version", "~> 1.2.0"},
		{"= 1.2.3", "version", "= 1.2.3"},
		{"!= 1.2.3", "version", "!= 1.2.3"},
		{"  >= 1.2.0  ", "version", ">= 1.2.0"},
		{">= banana", "", ""},
		{"^1.2.0", "", ""},
		{">=1.0 || <2.0", "", ""},
		{"refs/heads/main", "", ""},
		{"abcdef1", "", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			ref, version, err := ParseSourceRef(tc.input)
			if tc.field == "" {
				if err == nil || ref != "" || version != "" {
					t.Fatal(ref, version, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.field == "ref" && (ref != tc.value || version != "") || tc.field == "version" && (version != tc.value || ref != "") {
				t.Fatal(ref, version)
			}
		})
	}
}
