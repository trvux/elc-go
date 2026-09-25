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
	if got := OrDefault("", Center); got != Center {
		t.Errorf("OrDefault(\"\", Center) = %q, want %q", got, Center)
	}
	if got := OrDefault("", Left); got != Left {
		t.Errorf("OrDefault(\"\", Left) = %q, want %q", got, Left)
	}
	if got := OrDefault(Right, Center); got != Right {
		t.Errorf("OrDefault(%q, Center) = %q, want %q", Right, got, Right)
	}
	if got := OrDefault("diagonal", Center); got != "diagonal" {
		t.Errorf("OrDefault should pass invalid values through unchanged for the caller to reject, got %q", got)
	}
}
