package cmd

import "testing"

func TestParseProblemArg(t *testing.T) {
	tests := []struct {
		raw       string
		wantC, wP string
	}{
		{"c", "", "c"},
		{"C", "", "c"},
		{"3c", "abc3", "c"},
		{"455c", "abc455", "c"},
		{"abc455c", "abc455", "c"},
		{"ABC455C", "abc455", "c"},
		{"arc183f", "arc183", "f"},
		{"agc065a", "agc065", "a"},
		{"abc-455-c", "", "abc-455-c"}, // unparseable, passthrough
		{"abc", "", "abc"},              // letters only, no digits/letter form
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			gotC, gotP := parseProblemArg(tt.raw)
			if gotC != tt.wantC || gotP != tt.wP {
				t.Errorf("parseProblemArg(%q) = (%q, %q), want (%q, %q)",
					tt.raw, gotC, gotP, tt.wantC, tt.wP)
			}
		})
	}
}
