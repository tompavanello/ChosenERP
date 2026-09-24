package members

import "testing"

func TestInverseKind(t *testing.T) {
	cases := map[string]string{
		"parent":    "child",
		"child":     "parent",
		"discipler": "disciple",
		"disciple":  "discipler",
		"spouse":    "spouse",
		"dependent": "dependent",
		"relative":  "relative",
	}
	for in, want := range cases {
		if got := inverseKind(in); got != want {
			t.Errorf("inverseKind(%q)=%q, queria %q", in, got, want)
		}
	}
}
