package gql

import "testing"

func TestGameNameToSlug(t *testing.T) {
	cases := map[string]string{
		"Just Chatting": "just-chatting",
		"Tom Clancy's":  "tom-clancys",
		"Standoff 2":    "standoff-2",
	}
	for in, want := range cases {
		if got := GameNameToSlug(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
