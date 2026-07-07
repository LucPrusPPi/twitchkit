package client

import "testing"

func TestPickTopStream(t *testing.T) {
	streams := []StreamInfo{
		{UserLogin: "a", ViewerCount: 10},
		{UserLogin: "b", ViewerCount: 50},
		{UserLogin: "c", ViewerCount: 20},
	}
	got := PickTopStream(streams, "")
	if got == nil || got.UserLogin != "b" {
		t.Fatalf("want b, got %#v", got)
	}
	got = PickTopStream(streams, "b")
	if got == nil || got.UserLogin != "c" {
		t.Fatalf("skip b -> want c, got %#v", got)
	}
}
