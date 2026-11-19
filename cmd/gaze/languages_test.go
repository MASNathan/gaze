package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLanguages(t *testing.T) {
	testConfigDir(t)
	dir := t.TempDir()
	files := map[string]string{
		"a.GO":    "package example\nfunc Exported() {}\n",
		"b.mjs":   "export function exported() {}\n",
		"c.py":    "def exported():\n    pass\n",
		"d.ts":    "export interface Item { value: string }\n",
		"e.tsx":   "export function Item() { return <div />; }\n",
		"f.rs":    "pub fn item() {}\n",
		"g.java":  "public class Item { public void show() {} }\n",
		"h.kt":    "class Item { fun show() {} }\n",
		"i.cs":    "public class Item { public void Show() {} }\n",
		"j.c":     "int item(void) { return 1; }\n",
		"k.cpp":   "class Item { public: void show() {} };\n",
		"l.rb":    "def item\n  1\nend\n",
		"n.sh":    "item() { echo hi; }\n",
		"p.dart":  "class Item { void show() {} }\n",
		"q.scala": "class Item { def show(): Int = 1 }\n",
		"r.ex":    "defmodule Item do\n  def show(), do: 1\nend\n",
		"s.lua":   "function item() return 1 end\n",
		"t.R":     "item <- function(x) x\n",
		"u.zig":   "pub fn item() void {}\n",
		"v.m":     "@interface Item\n- (void)show;\n@end\n",
		"w.html":  "<main><h1>Example</h1></main>\n",
		"x.css":   ".item { color: red; --size: 1px; }\n",
	}
	for name, source := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		if code := run([]string{path}, &out, &stderr); code != 0 || out.Len() == 0 {
			t.Fatalf("%s: exit %d, output %q, error %q", name, code, out.String(), stderr.String())
		}
	}
	var out, stderr bytes.Buffer
	if code := run([]string{dir}, &out, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	for name := range files {
		if !strings.Contains(out.String(), name+":") {
			t.Errorf("missing %s in %q", name, out.String())
		}
	}
}
