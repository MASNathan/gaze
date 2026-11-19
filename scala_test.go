package gaze

import (
	"strings"
	"testing"
)

func TestScalaOutlineOptions(t *testing.T) {
	src := []byte("/** service */\nclass Service {\n  private def hidden(x: Int): Int = x\n  def run(x: Int): Int = { x + 1 }\n  val name: String = \"secret body\"\n}\n")
	got, err := scalaOutlineOptions(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "class Service") || !strings.Contains(got, "def run(x: Int): Int") || strings.Contains(got, "hidden") || strings.Contains(got, "secret body") || !strings.Contains(got, "service") {
		t.Fatalf("unexpected Scala outline:\n%s", got)
	}
	got, err = scalaOutlineOptions(src, Options{IncludeNonPublic: true, NoDocs: true, NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "hidden") || strings.Contains(got, "service") || strings.Contains(got, "L2 ") {
		t.Fatalf("unexpected Scala option output:\n%s", got)
	}
}
