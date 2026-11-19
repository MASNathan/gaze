package gaze

import (
	"strings"
	"testing"
)

func TestTypeScriptOutline(t *testing.T) {
	for _, tsx := range []bool{false, true} {
		body := "x.toString()"
		if tsx {
			body = "<div>{x}</div>"
		}
		source := []byte("/** API */\nexport interface User { name: string; run(x: number): void }\nexport class Service { private secret: number = 1; public run(x: number): string { return 'hidden' } }\nexport const render = (x: number) => " + body + ";\n")
		got, err := typescriptOutlineOptions(source, Options{NoLines: true}, tsx)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"API", "export interface User", "name: string", "run(x: number): void", "export class Service", "public run(x: number): string", "export const render"} {
			if !strings.Contains(got, want) {
				t.Errorf("tsx=%v missing %q: %s", tsx, want, got)
			}
		}
		for _, unwanted := range []string{"secret", "hidden", "<div>"} {
			if strings.Contains(got, unwanted) {
				t.Errorf("tsx=%v contains %q: %s", tsx, unwanted, got)
			}
		}
	}
}
