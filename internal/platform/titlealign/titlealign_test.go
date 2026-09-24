package titlealign

import "testing"

func TestValid(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{Left, true},
		{Center, true},
		{Right, true},
		{"", false},
		{"diagonal", false},
	}
	for _, c := range cases {
		if got := Valid(c.value); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.value, got, c.want)
		}
	}
}

func TestOrDefault(t *testing.T) {
	if got := OrDefault(""); got != Left {
		t.Errorf("OrDefault(\"\") = %q, want %q", got, Left)
	}
	if got := OrDefault(Right); got != Right {
		t.Errorf("OrDefault(%q) = %q, want %q", Right, got, Right)
	}
	if got := OrDefault("diagonal"); got != "diagonal" {
		t.Errorf("OrDefault should pass invalid values through unchanged for the caller to reject, got %q", got)
	}
}
