package main

import "testing"

func TestNormalizeBool(t *testing.T) {
	for _, test := range []struct {
		input int32
		want  int32
	}{
		{-1, 1},
		{0, 0},
		{1, 1},
		{42, 1},
	} {
		if got := normalizeBool(test.input); got != test.want {
			t.Errorf("normalizeBool(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}

func TestRGBBytesUseGRBOrder(t *testing.T) {
	got := rgbBytes(0x123456)
	want := [3]byte{0x34, 0x12, 0x56}
	if got != want {
		t.Fatalf("rgbBytes = %x, want %x", got, want)
	}
}
