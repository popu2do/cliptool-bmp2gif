package core

import "testing"

func TestGifOptionsNormalized(t *testing.T) {
	tests := []struct {
		name string
		in   GifOptions
		want GifOptions
	}{
		{name: "default", in: GifOptions{}, want: GifOptions{DelayMS: DefaultDelayMS}},
		{name: "minimum", in: GifOptions{DelayMS: 1}, want: GifOptions{DelayMS: MinDelayMS}},
		{name: "maximum", in: GifOptions{DelayMS: 5000}, want: GifOptions{DelayMS: MaxDelayMS}},
		{name: "valid", in: GifOptions{DelayMS: 750}, want: GifOptions{DelayMS: 750}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Normalized()
			if got != tt.want {
				t.Fatalf("Normalized() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGifOptionsDelayUnits(t *testing.T) {
	got := (GifOptions{DelayMS: 500}).DelayUnits()
	if got != 50 {
		t.Fatalf("DelayUnits() = %d, want 50", got)
	}
}
