package tokens

import (
	"sync"
	"testing"
)

func TestCount(t *testing.T) {
	// Token counts from github.com/tiktoken-go/tokenizer's o200k_base tests.
	for _, tc := range []struct {
		text string
		want int
	}{
		{"", 0},
		{"hello world", 2},
		{"hello  world", 3},
		{"supercalifragilistic", 6},
		{"We know what we are, but know not what we may be.", 14},
	} {
		t.Run(tc.text, func(t *testing.T) {
			got, err := Count([]byte(tc.text))
			if err != nil || got != tc.want {
				t.Fatalf("Count(%q) = %d, %v; want %d", tc.text, got, err, tc.want)
			}
		})
	}
	if _, err := Count([]byte{0xff}); err == nil {
		t.Fatal("Count accepted invalid UTF-8")
	}
}

func TestCountConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := Count([]byte("hello world")); err != nil || got != 2 {
				t.Errorf("Count = %d, %v; want 2", got, err)
			}
		}()
	}
	wg.Wait()
}
