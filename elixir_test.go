package gaze

import (
	"strings"
	"testing"
)

func TestElixirOutlineOptions(t *testing.T) {
	src := []byte("defmodule Service do\n  @moduledoc \"service\"\n  @doc \"runs\"\n  @spec run(integer()) :: integer()\n  def run(x) do\n    x + 1\n  end\n  defp hidden(x), do: x\nend\n")
	got, err := elixirOutlineOptions(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "defmodule Service") || !strings.Contains(got, "@spec run(integer()) :: integer()") || !strings.Contains(got, "def run(x)") || strings.Contains(got, "hidden") || !strings.Contains(got, "service") {
		t.Fatalf("unexpected Elixir outline:\n%s", got)
	}
	got, err = elixirOutlineOptions(src, Options{IncludeNonPublic: true, NoDocs: true, NoLines: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "defp hidden(x)") || strings.Contains(got, "service") || strings.Contains(got, "L1 ") {
		t.Fatalf("unexpected Elixir option output:\n%s", got)
	}
}
