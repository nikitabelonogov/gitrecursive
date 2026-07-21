package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindRepos(t *testing.T) {
	root := t.TempDir()
	mk := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, p, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk(".")
	mk("a")
	mk("b/c")
	mk("b/c/nested")
	if err := os.MkdirAll(filepath.Join(root, "plain/dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	abs := func(ps ...string) []string {
		out := make([]string, len(ps))
		for i, p := range ps {
			out[i] = filepath.Join(root, p)
		}
		return out
	}

	cases := []struct {
		depth int
		want  []string
	}{
		{-1, abs(".", "a", "b/c", "b/c/nested")},
		{0, abs(".")},
		{1, abs(".", "a")},
		{2, abs(".", "a", "b/c")},
	}
	for _, c := range cases {
		got, err := findRepos(root, c.depth)
		if err != nil {
			t.Fatalf("depth %d: %v", c.depth, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("depth %d: got %v, want %v", c.depth, got, c.want)
		}
	}
}

func TestFindReposEmpty(t *testing.T) {
	root := t.TempDir()
	got, err := findRepos(root, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no repos, got %v", got)
	}
}
