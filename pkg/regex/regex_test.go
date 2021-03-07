package regex_test

import (
	"regex/pkg/regex"
	"testing"
)

func TestRegex(t *testing.T) {
	r := regex.Compile("a(b|cd|ef)g")
	assertMatch(t, r, "abg")
	assertMatch(t, r, "acdg")
	assertMatch(t, r, "aefg")
	refuteMatch(t, r, "efg")
	refuteMatch(t, r, "aef")
	refuteMatch(t, r, "acd")
	refuteMatch(t, r, "ab")
	refuteMatch(t, r, "ef")
}

func assertMatch(t *testing.T, r regex.Regex, str string) {
	if !regex.Match(r, str) {
		t.Errorf("Expected %#v to match", str)
	}
}

func refuteMatch(t *testing.T, r regex.Regex, str string) {
	if regex.Match(r, str) {
		t.Errorf("Expected %#v not to match", str)
	}
}
