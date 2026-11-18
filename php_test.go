package gaze

import (
	"strings"
	"testing"
)

func TestPHPOutline(t *testing.T) {
	source := []byte(`<?php
namespace Demo {
/** API
 * Description.
 */
final class Service extends Base implements Contract {
    /** visible */
    public function run(string $value): int { return 42; }
    protected function hidden(): void { secret(); }
    private const SECRET = 1;
    public const NAME = 'ok';
    private string $token;
}
/** helper */
function helper(): void { internal(); }
}
`)
	for _, all := range []bool{false, true} {
		got, err := phpOutline(source, all, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"namespace Demo", "API | Description.", "L6-13 final class Service extends Base implements Contract", "visible", "L8 fn run(string $value): int", "const NAME = …", "L15", "fn helper(): void"} {
			if !strings.Contains(got, want) {
				t.Errorf("all=%v: missing %q in:\n%s", all, want, got)
			}
		}
		for _, hidden := range []string{"return 42", "secret()", "internal()"} {
			if strings.Contains(got, hidden) {
				t.Errorf("all=%v: body %q leaked in:\n%s", all, hidden, got)
			}
		}
		for _, private := range []string{"protected fn hidden", "private const SECRET", "private string $token"} {
			if strings.Contains(got, private) != all {
				t.Errorf("all=%v: visibility mismatch for %q in:\n%s", all, private, got)
			}
		}
	}
}

func TestPHPOutlineOmitsExpressions(t *testing.T) {
	source := []byte(`<?php
#[Dangerous(secret())]
class Example {
    public $first = 123, $second = [1, 2];
    public const ONE = 1, TWO = 2;
    public function run($arg = new Thing()) { danger(); }
}
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"$first = …", "$second = …", "ONE = …", "TWO = …", "$arg = …"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"secret()", "123", "[1, 2]", "new Thing()", "danger()"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, got)
		}
	}
}

func TestPHPOutlineTopLevel(t *testing.T) {
	source := []byte(`<?php
namespace A;
use B\C;
const LIMIT = 3;
interface I { function go(): void; }
trait T { private function x() {} }
enum Status: string { case Ready = 'ready'; }
function outer() { function nested() {} }
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"namespace A", "const LIMIT = …", "interface I", "fn go(): void", "trait T", "enum Status: string", "case Ready = …", "fn outer()"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"private fn x", "fn nested", `use B\C;`, "'ready'"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, got)
		}
	}
}

func TestPHPOutlineDocModes(t *testing.T) {
	source := []byte(`<?php
class Example {
    /**
     * {@inheritDoc}
     * Keeps  spaced text and a|b.
     * @param array<string, int> $items description
     * @return Collection<int> values
     */
    public function run(array $items): iterable {}
    /** @var list<int> */
    public array $values;
}
`)
	compact, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"L2-12 class Example", "L9 fn run(array $items): iterable", "L11 array $values", "Keeps  spaced text and a|b.", "@param array<string, int> $items description", "@return Collection<int> values", "@var list<int>"} {
		if !strings.Contains(compact, want) {
			t.Errorf("compact missing %q in:\n%s", want, compact)
		}
	}
	full, err := phpOutline(source, false, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/**\n     * {@inheritDoc}\n", "public function run(array $items): iterable", "public array $values;"} {
		if !strings.Contains(full, want) {
			t.Errorf("full missing %q in:\n%s", want, full)
		}
	}
}

func TestPHPCompactDocTypes(t *testing.T) {
	source := []byte(`<?php
/**
 * Summary {@inheritDoc} remains.
 * @param string $a
 * @param int $b description
 * @param array{foo: list<int>, bar: string} $c shape
 * @param string $missing
 * @return bool
 * @throws Exception failure
 * @custom keep this
 */
function example(string $a, int $b, array $c): bool {}
/**
 * @param string $x
 * @return list<int> values
 * @return string another
 */
function undocumented($x) {}
/**
 * @param array<
 *   string, list<int>
 * > $items multiline
 * @return int
 *   description continued
 */
function multiline(array $items): int {}
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Summary  remains.", "@param int $b description", "@param array{foo: list<int>, bar: string} $c shape", "@param string $missing", "@throws Exception failure", "@custom keep this", "string $x", "@return list<int> values", "@return string another", "@param array<", "string, list<int>", "> $items multiline", "@return int", "description continued"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"@param string $a", "@return bool", "{@inheritDoc}", "/**"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, got)
		}
	}
	full, err := phpOutline(source, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full, "* @param string $a\n") || strings.Contains(full, "doc<string>") {
		t.Errorf("full changed:\n%s", full)
	}
}

func TestPHPCompactExample(t *testing.T) {
	source := []byte("<?php\n/**\n * @param string $query\n * @param array $binds\n * @param string $entityClass\n * @return array\n * @throws Exception in case of error\n */\nfunction query($query, array $binds = [], $entityClass = NULL) {}\n")
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	want := "@throws Exception in case of error\nL9 fn query(string $query, array $binds = [], string $entityClass = null): array\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPHPCompactReferencesAndDefaults(t *testing.T) {
	source := []byte("<?php\n/**\n * @param int $ref description\n * @param string $rest\n */\nfunction f(&$ref, &...$rest) {}\nfunction g($a = TRUE, $b = FALSE, $c = NULL, $d = [], $e = make()) {}\n")
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"@param int $ref description", "fn f(int &$ref, string &...$rest)", "fn g($a = true, $b = false, $c = null, $d = [], $e = …)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestPHPKeywordToken(t *testing.T) {
	source := []byte("<?php\nclass X {\n #[Attr('function')]\n public /* function comment */ function\tfunctionName($x = 'function') {}\n}\n")
	for _, tc := range []struct {
		opts Options
		want string
	}{
		{Options{}, "/* function comment */ fn\tfunctionName($x = 'function')"},
		{Options{FullDocs: true}, "/* function comment */ function\tfunctionName($x = …)"},
	} {
		got, err := Outline("x.php", source, tc.opts)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("missing %q in %q", tc.want, got)
		}
	}
}

func TestPHPCompactTypeIdentityAndSemicolons(t *testing.T) {
	source := []byte(`<?php
/**
 * @param  \Closure  $callback
 * @param \Closure $described explanation
 * @param Other $conflict
 * @param Missing $absent
 */
function run(Closure $callback, Closure $described, Closure $conflict, $text = ";") {}
interface I { function multi(
 string $text = ";"
): void; }
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`@param \Closure $described explanation`, "@param Other $conflict", "@param Missing $absent", `$text = …`, `): void`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "$callback |") || strings.Contains(got, `@param  \Closure  $callback`) {
		t.Errorf("redundant tag retained:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasSuffix(line, ";") {
			t.Errorf("trailing semicolon: %q", line)
		}
	}
	full, err := phpOutline(source, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full, "): void;") {
		t.Errorf("full declaration lost semicolon:\n%s", full)
	}
}

func TestPHPDocImportedIdentity(t *testing.T) {
	source := []byte(`<?php
namespace App\CodeIgniter;
use Closure;
/** @param  \Closure  $callback */
function remember(Closure $callback) {}
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "@param") || !strings.Contains(got, "Closure $callback") {
		t.Errorf("redundant imported type retained: %s", got)
	}
}

func TestPHPDocNamespacedIdentity(t *testing.T) {
	source := []byte(`<?php
namespace Demo;
/** @param \Closure $callback */
function run(Closure $callback) {}
`)
	got, err := phpOutline(source, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `@param \Closure $callback`) {
		t.Errorf("ambiguous type discarded:\n%s", got)
	}
	if strings.Contains(got, "namespace Demo;") {
		t.Errorf("namespace semicolon retained:\n%s", got)
	}
}
