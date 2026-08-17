package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

type testCase struct {
	name string
	in   string
	want string
}

// Cases where lines should be joined.
var joinCases = []testCase{
	{"two wrapped lines", "one two\nthree four\n", "one two three four\n"},
	{"five wrapped lines", "a\nb\nc\nd\ne\n", "a b c d e\n"},
	{"no final newline", "one\ntwo", "one two"},
	{"single trailing space joins", "one \ntwo\n", "one two\n"},
	{"trailing tab joins", "one\t\ntwo\n", "one two\n"},
	{"even backslashes are literal", "one\\\\\ntwo\n", "one\\\\ two\n"},
	{"continuation indent collapses", "one\n    two\n", "one two\n"},
	{"hash without space is not a heading", "#foo\nbar\n", "#foo bar\n"},
	{"seven hashes is not a heading", "####### foo\nbar\n", "####### foo bar\n"},
	{"two dashes is not a break", "one\n--x\n", "one --x\n"},
	{"emphasis at line start", "one\n*emphasis*\n", "one *emphasis*\n"},
	{"pipe text that is not a table", "one | two\nthree | four\n", "one | two three | four\n"},
	{"setext content joins", "Title text\nmore title\n=====\n", "Title text more title\n=====\n"},
	{"list item wraps", "- item text\n  wrapped more\n", "- item text wrapped more\n"},
	{"list lazy continuation", "- item text\nlazy cont\n", "- item text lazy cont\n"},
	{"ordered list paren", "1) item text\n   wrapped\n", "1) item text wrapped\n"},
	{"ordered list dot", "1. item text\n   wrapped\n", "1. item text wrapped\n"},
	{"double digit ordered", "10. item text\n    wrapped\n", "10. item text wrapped\n"},
	{"plus marker", "+ item text\n  wrapped\n", "+ item text wrapped\n"},
	{"star marker", "* item text\n  wrapped\n", "* item text wrapped\n"},
	{"task item wraps", "- [ ] task text\n  wrapped\n", "- [ ] task text wrapped\n"},
	{"blockquote wraps", "> one two\n> three four\n", "> one two three four\n"},
	{"blockquote no space", ">one\n>two\n", ">one two\n"},
	{"blockquote lazy", "> one\ntwo\n", "> one two\n"},
	{"nested quote wraps", "> > one\n> > two\n", "> > one two\n"},
	{"quoted list wraps", "> - item one\n>   wrapped\n", "> - item one wrapped\n"},
	{"admonition body joins", ":::note\nBody text\nmore body\n:::\n", ":::note\nBody text more body\n:::\n"},
	{"definition wraps", "Term\n: definition that\n  wraps\n", "Term\n: definition that wraps\n"},
}

// Cases where the input must be returned unchanged.
var preserveCases = []testCase{
	{"empty", "", ""},
	{"single line", "one\n", "one\n"},
	{"only newline", "\n", "\n"},
	{"blank line between paragraphs", "one\n\ntwo\n", "one\n\ntwo\n"},
	{"multiple blank lines", "one\n\n\n\ntwo\n", "one\n\n\n\ntwo\n"},
	{"whitespace-only line preserved", "one\n   \ntwo\n", "one\n   \ntwo\n"},
	{"leading and trailing blanks", "\n\none\n\n\n", "\n\none\n\n\n"},
	{"two space hard break", "one  \ntwo\n", "one  \ntwo\n"},
	{"three space hard break", "one   \ntwo\n", "one   \ntwo\n"},
	{"backslash hard break", "one\\\ntwo\n", "one\\\ntwo\n"},
	{"hard break in list", "- item  \n  next\n", "- item  \n  next\n"},
	{"hard break in quote", "> one  \n> two\n", "> one  \n> two\n"},
	{"code span spans lines", "a `code  \nspan` b\n", "a `code  \nspan` b\n"},
	{"atx heading then text", "# Heading\ntext\n", "# Heading\ntext\n"},
	{"text then atx heading", "text\n# Heading\n", "text\n# Heading\n"},
	{"closed atx heading", "# Heading #\ntext\n", "# Heading #\ntext\n"},
	{"setext equals", "Title\n=====\ntext\n", "Title\n=====\ntext\n"},
	{"setext dashes", "Title\n-----\ntext\n", "Title\n-----\ntext\n"},
	{"thematic break dashes", "one\n\n---\n\ntwo\n", "one\n\n---\n\ntwo\n"},
	{"thematic break stars", "one\n\n***\n\ntwo\n", "one\n\n***\n\ntwo\n"},
	{"thematic break underscores", "one\n\n___\n\ntwo\n", "one\n\n___\n\ntwo\n"},
	{"spaced thematic break", "one\n\n- - -\n\ntwo\n", "one\n\n- - -\n\ntwo\n"},
	{"fenced code backtick", "```\nfoo\nbar\n```\n", "```\nfoo\nbar\n```\n"},
	{"fenced code tilde", "~~~\nfoo\nbar\n~~~\n", "~~~\nfoo\nbar\n~~~\n"},
	{"fenced code with info", "```go\nfoo\nbar\n```\n", "```go\nfoo\nbar\n```\n"},
	{"fenced code longer closer", "```\nfoo\nbar\n`````\n", "```\nfoo\nbar\n`````\n"},
	{"unclosed fence", "```\nfoo\nbar\n", "```\nfoo\nbar\n"},
	{"fence in list item", "- item\n\n  ```\n  foo\n  bar\n  ```\n", "- item\n\n  ```\n  foo\n  bar\n  ```\n"},
	{"fence in blockquote", "> ```\n> foo\n> bar\n> ```\n", "> ```\n> foo\n> bar\n> ```\n"},
	{"indented code", "para\n\n    foo\n    bar\n", "para\n\n    foo\n    bar\n"},
	{"indented code with blank", "para\n\n    foo\n\n    bar\n", "para\n\n    foo\n\n    bar\n"},
	{"sibling list items", "- one\n- two\n", "- one\n- two\n"},
	{"nested list", "- one\n  - nested\n", "- one\n  - nested\n"},
	{"loose list", "- one\n\n- two\n", "- one\n\n- two\n"},
	{"quote depth increase", "> one\n> > two\n", "> one\n> > two\n"},
	{"blank quote line", "> one\n>\n> two\n", "> one\n>\n> two\n"},
	{"table basic", "a | b\n--|--\n1 | 2\n", "a | b\n--|--\n1 | 2\n"},
	{"table piped", "| a | b |\n|---|---|\n| 1 | 2 |\n", "| a | b |\n|---|---|\n| 1 | 2 |\n"},
	{"table aligned", "| a | b | c |\n|:--|--:|:-:|\n| 1 | 2 | 3 |\n", "| a | b | c |\n|:--|--:|:-:|\n| 1 | 2 | 3 |\n"},
	{"para above table", "text\n\na | b\n--|--\n1 | 2\n", "text\n\na | b\n--|--\n1 | 2\n"},
	{"table escaped pipe", "| a | b |\n|---|---|\n| x \\| y | 2 |\n", "| a | b |\n|---|---|\n| x \\| y | 2 |\n"},
	{"yaml frontmatter", "---\na: 1\nb: 2\n---\n\ntext\n", "---\na: 1\nb: 2\n---\n\ntext\n"},
	{"yaml frontmatter dots", "---\na: 1\nb: 2\n...\n\ntext\n", "---\na: 1\nb: 2\n...\n\ntext\n"},
	{"toml frontmatter", "+++\na = 1\nb = 2\n+++\n\ntext\n", "+++\na = 1\nb = 2\n+++\n\ntext\n"},
	{"frontmatter wrapped value", "---\na: one\n  two\n---\n", "---\na: one\n  two\n---\n"},
	{"html comment multiline", "<!--\nfoo\nbar\n-->\n", "<!--\nfoo\nbar\n-->\n"},
	{"html comment oneline", "<!-- foo -->\n", "<!-- foo -->\n"},
	{"html div block", "<div>\n  foo\n  bar\n</div>\n", "<div>\n  foo\n  bar\n</div>\n"},
	{"html pre block", "<pre>\nfoo\n\nbar\n</pre>\n", "<pre>\nfoo\n\nbar\n</pre>\n"},
	{"html script block", "<script>\nvar a = 1\nvar b = 2\n</script>\n", "<script>\nvar a = 1\nvar b = 2\n</script>\n"},
	{"html doctype", "<!DOCTYPE html>\n<div>\nfoo\n</div>\n", "<!DOCTYPE html>\n<div>\nfoo\n</div>\n"},
	{"html cdata", "<![CDATA[\nfoo\nbar\n]]>\n", "<![CDATA[\nfoo\nbar\n]]>\n"},
	{"link ref def", "[a]: /url\n[b]: /url2\n", "[a]: /url\n[b]: /url2\n"},
	{"link ref def with title", "[a]: /url \"title\"\n", "[a]: /url \"title\"\n"},
	{"math block", "$$\nx = 1\ny = 2\n$$\n", "$$\nx = 1\ny = 2\n$$\n"},
	{"obsidian comment", "%% one\n%% two\n", "%% one\n%% two\n"},
	{"mdx expression", "{/* one */}\n{/* two */}\n", "{/* one */}\n{/* two */}\n"},
	{"admonition delimiters", ":::note\n:::\n", ":::note\n:::\n"},
}

func TestUnwrap(t *testing.T) {
	for _, group := range []struct {
		label string
		cases []testCase
	}{
		{"join", joinCases},
		{"preserve", preserveCases},
	} {
		for _, tc := range group.cases {
			t.Run(group.label+"/"+tc.name, func(t *testing.T) {
				got, err := Unwrap([]byte(tc.in))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != tc.want {
					t.Errorf("expected %q; got %q", tc.want, got)
				}
				checkInvariants(t, tc.in, string(got))
			})
		}
	}
}

// TestCRLF checks that line terminators survive, since they live in the
// regions copied through verbatim.
func TestCRLF(t *testing.T) {
	cases := []testCase{
		{"crlf join", "one\r\ntwo\r\n", "one two\r\n"},
		{"crlf preserved on separate blocks", "one\r\n\r\ntwo\r\n", "one\r\n\r\ntwo\r\n"},
		{"crlf hard break", "one  \r\ntwo\r\n", "one  \r\ntwo\r\n"},
		{"crlf no final newline", "one\r\ntwo", "one two"},
		{"mixed endings", "one\r\ntwo\n\nthree\r\n", "one two\n\nthree\r\n"},
		{"crlf frontmatter", "---\r\na: 1\r\n---\r\n\r\ntext\r\n", "---\r\na: 1\r\n---\r\n\r\ntext\r\n"},
		{"crlf table", "a | b\r\n--|--\r\n1 | 2\r\n", "a | b\r\n--|--\r\n1 | 2\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Unwrap([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("expected %q; got %q", tc.want, got)
			}
			checkInvariants(t, tc.in, string(got))
		})
	}
}

// TestUnclosedFrontmatter pins the choice to treat an unterminated leading
// "---" as a thematic break rather than disabling the tool for the whole file.
func TestUnclosedFrontmatter(t *testing.T) {
	got, err := Unwrap([]byte("---\none two\nthree four\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\none two three four\n"; string(got) != want {
		t.Errorf("expected %q; got %q", want, got)
	}
}

// TestFootnotes covers the extension whose AST transformer moves definitions to
// the end of the document, so collected ranges must be sorted by source offset.
func TestFootnotes(t *testing.T) {
	in := "Text with a ref[^1] and more\nwrapped prose here.\n\n[^1]: note text\n    wrapped note\n"
	want := "Text with a ref[^1] and more wrapped prose here.\n\n[^1]: note text wrapped note\n"
	got, err := Unwrap([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("expected %q; got %q", want, got)
	}
}

func TestGolden(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, in := range inputs {
		if !strings.HasSuffix(in, ".expected.md") {
			names = append(names, in)
		}
	}
	if len(names) == 0 {
		t.Fatal("no testdata fixtures found")
	}

	for _, path := range names {
		t.Run(filepath.Base(path), func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expectedPath := strings.TrimSuffix(path, ".md") + ".expected.md"
			want, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Unwrap(src)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("output does not match %s\n%s", expectedPath, lineDiff(string(want), string(got)))
			}
			checkInvariants(t, string(src), string(got))
		})
	}
}

// TestNoJoinFixtureUnchanged asserts the strongest single regression property:
// a document with nothing joinable comes back byte-identical.
func TestNoJoinFixtureUnchanged(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "nojoin.md"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unwrap(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, src) {
		t.Errorf("nojoin.md was modified\n%s", lineDiff(string(src), string(got)))
	}
}

func TestIdempotent(t *testing.T) {
	var all []testCase
	all = append(all, joinCases...)
	all = append(all, preserveCases...)
	for _, tc := range all {
		t.Run(tc.name, func(t *testing.T) {
			once, err := Unwrap([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			twice, err := Unwrap(once)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(once, twice) {
				t.Errorf("not idempotent: %q then %q", once, twice)
			}
		})
	}
}

func FuzzUnwrap(f *testing.F) {
	for _, tc := range append(append([]testCase{}, joinCases...), preserveCases...) {
		f.Add(tc.in)
	}
	if paths, err := filepath.Glob(filepath.Join("testdata", "*.md")); err == nil {
		for _, p := range paths {
			if b, err := os.ReadFile(p); err == nil {
				f.Add(string(b))
			}
		}
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, err := Unwrap([]byte(in))
		if err != nil {
			t.Fatal(err)
		}
		checkInvariants(t, in, string(got))

		twice, err := Unwrap(got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, twice) {
			t.Fatalf("not idempotent for %q: %q then %q", in, got, twice)
		}
	})
}

// checkInvariants asserts the properties that must hold for every input: no
// non-whitespace content may change, the output may not grow, and the rendered
// HTML must be equivalent.
func checkInvariants(t *testing.T, in, out string) {
	t.Helper()
	if contentOnly(in) != contentOnly(out) {
		t.Errorf("content changed\n in: %q\nout: %q", contentOnly(in), contentOnly(out))
	}
	if len(out) > len(in) {
		t.Errorf("output grew from %d to %d bytes", len(in), len(out))
	}
	if a, b := renderNormalized([]byte(in)), renderNormalized([]byte(out)); a != b {
		t.Errorf("rendered output differs\n in: %q\nout: %q", a, b)
	}
}

// contentOnly reduces a document to the characters a join must never change:
// everything except whitespace and blockquote markers. Markers are excluded
// because joining two quoted lines legitimately drops the second line's ">",
// which is container syntax rather than content. Comparing this before and
// after is a cheap oracle for dropped or duplicated text.
func contentOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) && r != '>' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			fmt.Fprintf(&b, "line %d:\n  want %q\n   got %q\n", i+1, wl, gl)
		}
	}
	return b.String()
}
