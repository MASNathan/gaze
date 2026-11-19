package gaze

import (
	"strings"
	"testing"
)

func TestZigOutlineOptions(t *testing.T) {
	src := []byte("/// Adds values\npub fn add(a: i32, b: i32) i32 { return a + b; }\nfn hidden() void {}\n")
	out, err := zigOutlineOptions(src, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pub fn add(a: i32, b: i32) i32") || strings.Contains(out, "return") || strings.Contains(out, "hidden") {
		t.Fatal(out)
	}
	out, err = zigOutlineOptions(src, Options{NoLines: true, NoDocs: true, IncludeNonPublic: true})
	if err != nil || !strings.Contains(out, "hidden") || strings.Contains(out, "Adds values") {
		t.Fatalf("%s %v", out, err)
	}
}

func TestObjcOutlineOptions(t *testing.T) {
	src := []byte("@interface Thing : NSObject\n@property int count;\n- (int)value;\n@end\nint f(void) { return 42; }\n")
	out, err := objcOutlineOptions(src, Options{NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@interface Thing") || !strings.Contains(out, "@property int count;") || !strings.Contains(out, "- (int)value;") || strings.Contains(out, "return 42") {
		t.Fatal(out)
	}
}
