package main

import (
	"bytes"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Lines that look like prose to goldmark but actually carry structure in some
// widely used Markdown dialect. We never join across one of these.
//
//	:::note   Docusaurus / MkDocs admonitions and generic directives
//	$$        display math
//	%%        Obsidian comments
//	{...}     MDX expressions
//	[!NOTE]   GitHub alerts, which only render when alone on their line
var guardLine = regexp.MustCompile(`^(:{3,}|\$\$|%%|\{|\[!\w+\]$)`)

// blockStart matches a line opening a list item, ATX heading, blockquote, or
// definition description. Each of these markers is recognized only when
// followed by whitespace or end of line, so appending to a line can bring one
// into being where there was none: "*" and "000" are two lines of prose, but
// "* 000" is a list item.
//
// A line already matching this would have been a block start rather than
// paragraph content, so goldmark would never have offered it for joining. That
// makes testing the prospective line sufficient, with no false refusals for
// ordinary prose. Markers needing no following character — "<", "```", ">" —
// cannot be created this way, since a line bearing one already opens a block.
var blockStart = regexp.MustCompile(
	`^ {0,3}([-*+]([ \t]|$)|\d{1,9}[.)]([ \t]|$)|#{1,6}([ \t]|$)|:([ \t]|$)|>)`)

// isThematicBreak reports whether line is a horizontal rule: three or more of
// the same -, * or _, separated only by spaces or tabs. Joining can conjure one
// of these too, since a rule tolerates internal spaces.
func isThematicBreak(line []byte) bool {
	i, indent := 0, 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		indent++
		i++
	}
	if indent > 3 || i >= len(line) {
		return false
	}
	marker := line[i]
	if marker != '-' && marker != '*' && marker != '_' {
		return false
	}
	count := 0
	for ; i < len(line); i++ {
		switch line[i] {
		case marker:
			count++
		case ' ', '\t':
		default:
			return false
		}
	}
	return count >= 3
}

// joinRange is a source byte range to be replaced by a single space.
type joinRange struct{ start, stop int }

// newMarkdown returns the parser used both to find joinable blocks and to
// verify the result.
//
// GFM is required for correctness, not decoration: without the table extension
// a pipe table parses as an ordinary paragraph and we would collapse its rows
// into one line. Footnote and DefinitionList likewise turn constructs that
// would otherwise look like prose into real nodes.
func newMarkdown() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(
		extension.GFM,
		extension.Footnote,
		extension.DefinitionList,
	))
}

// Unwrap returns src with soft-wrapped lines joined into single lines.
//
// It uses goldmark only to locate the blocks whose line breaks are pure soft
// wrapping; every other byte of src is copied through untouched, so the result
// differs from the input only where lines were deliberately joined.
//
// The result is then verified to render identically to the input. Markdown has
// enough dialects and enough context-sensitive inline rules that no set of
// static checks catches every way a join could alter meaning, so rather than
// trust the analysis we confirm it, and drop any joins that do not hold up.
func Unwrap(src []byte) ([]byte, error) {
	groups, err := findJoins(src)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return append([]byte(nil), src...), nil
	}

	want := renderNormalized(src)

	var all []joinRange
	for _, g := range groups {
		all = append(all, g...)
	}
	if out := apply(src, all); renderNormalized(out) == want {
		return out, nil
	}

	// Something in this document does not survive being unwrapped. Re-admit the
	// blocks one at a time, keeping only those that preserve the rendering, so a
	// single awkward construct costs only its own joins.
	var kept []joinRange
	for _, g := range groups {
		trial := append(append([]joinRange(nil), kept...), g...)
		sort.Slice(trial, func(i, j int) bool { return trial[i].start < trial[j].start })
		if renderNormalized(apply(src, trial)) == want {
			kept = trial
		}
	}
	return apply(src, kept), nil
}

// findJoins returns the ranges to collapse, grouped by the block they belong to
// and ordered by position in src.
func findJoins(src []byte) ([][]joinRange, error) {
	// goldmark has no concept of frontmatter and would read the closing "---"
	// as a setext heading underline, so hold it aside and parse only the body.
	bodyStart := frontmatterEnd(src)
	body := src[bodyStart:]

	doc := newMarkdown().Parser().Parse(text.NewReader(body))

	// goldmark has no math extension, so a $$ block looks like an ordinary
	// paragraph. Its line breaks are significant LaTeX, so exclude it.
	math := mathRegions(src)

	var groups [][]joinRange
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !joinable(n) {
			return ast.WalkContinue, nil
		}
		var group []joinRange
		for _, r := range joinsIn(n, body) {
			r = joinRange{r.start + bodyStart, r.stop + bodyStart}
			if overlapsAny(math, r.start, r.stop) {
				continue
			}
			group = append(group, r)
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}

	// The footnote extension moves definitions to the end of the document, so
	// walk order does not follow source order.
	sort.Slice(groups, func(i, j int) bool { return groups[i][0].start < groups[j][0].start })
	return groups, nil
}

// renderNormalized renders src to HTML and collapses whitespace runs outside of
// <pre> and <code>, where whitespace is part of the content. A soft line break
// renders as a newline and a joined line renders as a space, so this
// normalization is what makes the two comparable, while still noticing a join
// that altered the literal text of a code span.
func renderNormalized(src []byte) string {
	var buf bytes.Buffer
	if err := newMarkdown().Convert(src, &buf); err != nil {
		return "render error: " + err.Error()
	}

	var b strings.Builder
	h := buf.Bytes()
	verbatim := 0
	pendingSpace := false
	for i := range h {
		if h[i] == '<' {
			switch {
			case hasPrefixAt(h, i, "<pre"), hasPrefixAt(h, i, "<code"):
				verbatim++
			case hasPrefixAt(h, i, "</pre"), hasPrefixAt(h, i, "</code"):
				if verbatim > 0 {
					verbatim--
				}
			}
		}
		if verbatim == 0 && isSpace(h[i]) {
			pendingSpace = true
			continue
		}
		if pendingSpace {
			b.WriteByte(' ')
			pendingSpace = false
		}
		b.WriteByte(h[i])
	}
	return strings.TrimSpace(b.String())
}

func hasPrefixAt(b []byte, i int, prefix string) bool {
	return bytes.HasPrefix(b[i:], []byte(prefix))
}

// joinable reports whether n's line breaks are soft wrapping we may remove.
// Every other node kind — code blocks, HTML blocks, tables, thematic breaks,
// and anything goldmark does not recognize — is left alone by omission.
func joinable(n ast.Node) bool {
	switch n.Kind() {
	case ast.KindParagraph, ast.KindTextBlock, ast.KindHeading:
		// A Heading has multiple lines only when it is a wrapped setext
		// heading; joining those renders identically.
	default:
		return false
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == extast.KindTable {
			return false
		}
	}
	return true
}

// joinsIn returns the ranges to collapse between n's consecutive lines.
//
// goldmark's line segments start after any container prefix (blockquote "> "
// markers, list marker and indentation) and run through the line terminator.
// So joining two lines is exactly "replace the span from the end of one line's
// content to the start of the next line's content with a single space", which
// leaves prefixes, markers, and indentation untouched.
func joinsIn(n ast.Node, src []byte) []joinRange {
	lines := n.Lines()
	count := lines.Len()
	if count < 2 {
		return nil
	}
	hard := hardBreaks(n)
	verbatim := verbatimInlines(n)

	// Content bounds of each line, excluding indentation and the terminator.
	start := make([]int, count)
	end := make([]int, count)
	for i := range count {
		s := lines.At(i)
		start[i] = contentStart(src, s)
		end[i] = contentEnd(src, s)
	}

	// canJoin[i] reports whether the break between line i and i+1 is pure
	// wrapping rather than something the author or the syntax requires.
	canJoin := make([]bool, count-1)
	for i := range canJoin {
		switch {
		case start[i] >= end[i], start[i+1] >= end[i+1]:
			// One side has no content.
		case endsWithHardBreak(hard, lines.At(i)):
			// An explicit hard break is a line break the author asked for.
		case overlapsAny(verbatim, end[i], start[i+1]):
			// Whitespace is significant inside a code span or an inline HTML
			// tag, so a break falling inside one is content, not wrapping.
		case guardLine.Match(src[start[i]:end[i]]), guardLine.Match(src[start[i+1]:end[i+1]]):
		default:
			canJoin[i] = true
		}
	}

	// Decide each run of joinable lines as a whole rather than one break at a
	// time. Whether a join is safe depends on the entire resulting line, so
	// judging the finished line keeps the result independent of how far joining
	// had progressed -- which is what makes the transform idempotent.
	var out []joinRange
	for i := 0; i < count-1; {
		if !canJoin[i] {
			i++
			continue
		}
		last := i
		for last < count-1 && canJoin[last] {
			last++
		}

		joined := append([]byte(nil), src[start[i]:end[i]]...)
		for k := i + 1; k <= last; k++ {
			joined = append(joined, ' ')
			joined = append(joined, src[start[k]:end[k]]...)
		}
		// Refuse the whole run if collapsing it would turn prose into a new
		// block: "*" and "000" are two lines of text, but "* 000" is a list
		// item, and "**" with "*" would become a horizontal rule.
		if !isThematicBreak(joined) && !blockStart.Match(joined) {
			for k := i; k < last; k++ {
				out = append(out, joinRange{end[k], start[k+1]})
			}
		}
		i = last
	}
	return out
}

// hardBreaks collects the source offsets at which n's inline content has an
// explicit hard line break. Taking this from the inline AST rather than
// scanning raw bytes gets the edge cases right: trailing spaces inside a
// multi-line code span are content, not a hard break.
func hardBreaks(n ast.Node) []int {
	var stops []int
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		_ = ast.Walk(c, func(in ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if t, ok := in.(*ast.Text); ok && t.HardLineBreak() {
				stops = append(stops, t.Segment.Stop)
			}
			return ast.WalkContinue, nil
		})
	}
	return stops
}

// verbatimInlines returns the source extents of n's inline nodes whose content
// is reproduced literally, so we can avoid joining a line break that lives
// inside one. A code span renders its internal whitespace, so collapsing
// "`a  \nb`" to "`a b`" would change the output.
func verbatimInlines(n ast.Node) []joinRange {
	var out []joinRange
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		_ = ast.Walk(c, func(in ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			switch in.Kind() {
			case ast.KindCodeSpan, ast.KindRawHTML, ast.KindAutoLink:
			default:
				return ast.WalkContinue, nil
			}
			if r, ok := inlineExtent(in); ok {
				out = append(out, r)
			}
			return ast.WalkSkipChildren, nil
		})
	}
	return out
}

// inlineExtent returns the source range spanned by an inline node, derived from
// the segments of its own text or of its descendants.
func inlineExtent(n ast.Node) (joinRange, bool) {
	start, stop := -1, -1
	note := func(a, b int) {
		if start < 0 || a < start {
			start = a
		}
		if b > stop {
			stop = b
		}
	}
	if h, ok := n.(*ast.RawHTML); ok && h.Segments != nil {
		for i := 0; i < h.Segments.Len(); i++ {
			s := h.Segments.At(i)
			note(s.Start, s.Stop)
		}
	}
	_ = ast.Walk(n, func(in ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := in.(*ast.Text); ok {
			note(t.Segment.Start, t.Segment.Stop)
		}
		return ast.WalkContinue, nil
	})
	if start < 0 {
		return joinRange{}, false
	}
	return joinRange{start, stop}, true
}

// overlapsAny reports whether [start, stop) intersects any of the ranges.
func overlapsAny(ranges []joinRange, start, stop int) bool {
	for _, r := range ranges {
		if start < r.stop && r.start < stop {
			return true
		}
	}
	return false
}

func endsWithHardBreak(stops []int, line text.Segment) bool {
	for _, s := range stops {
		if s >= line.Start && s <= line.Stop {
			return true
		}
	}
	return false
}

// contentEnd returns the offset just past the line's last non-blank byte,
// excluding the line terminator.
func contentEnd(src []byte, s text.Segment) int {
	end := min(s.Stop, len(src))
	for end > s.Start && isSpace(src[end-1]) {
		end--
	}
	return end
}

// contentStart returns the offset of the line's first non-blank byte.
func contentStart(src []byte, s text.Segment) int {
	start := s.Start
	for start < s.Stop && start < len(src) && isSpace(src[start]) {
		start++
	}
	return start
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// apply copies src, replacing each range with a single space. Ranges must be
// sorted by start offset.
func apply(src []byte, ranges []joinRange) []byte {
	if len(ranges) == 0 {
		return append([]byte(nil), src...)
	}
	out := make([]byte, 0, len(src))
	pos := 0
	for _, r := range ranges {
		if r.start < pos {
			continue // defensive: overlapping ranges should not happen
		}
		out = append(out, src[pos:r.start]...)
		out = append(out, ' ')
		pos = r.stop
	}
	return append(out, src[pos:]...)
}

// mathRegions returns the extents of "$$"-delimited display math blocks. An
// unterminated opener yields no region, matching how frontmatter is treated.
func mathRegions(src []byte) []joinRange {
	var out []joinRange
	openAt := -1
	for pos := 0; pos < len(src); {
		end := lineEnd(src, pos)
		// Trim only Markdown's whitespace, matching isSpace. Using
		// bytes.TrimSpace here would also strip form feeds and vertical tabs,
		// which are ordinary content, and the disagreement between the two
		// notions of blank would make the transform non-idempotent.
		if string(bytes.Trim(src[pos:end], " \t\r\n")) == "$$" {
			if openAt < 0 {
				openAt = pos
			} else {
				out = append(out, joinRange{openAt, end})
				openAt = -1
			}
		}
		pos = end
	}
	return out
}

// frontmatterEnd returns the offset just past a leading YAML or TOML
// frontmatter block, or 0 if src does not open with one. An unterminated
// opening delimiter is not frontmatter: it is an ordinary thematic break.
func frontmatterEnd(src []byte) int {
	var closers []string
	switch {
	case opensWith(src, "---"):
		closers = []string{"---", "..."}
	case opensWith(src, "+++"):
		closers = []string{"+++"}
	default:
		return 0
	}

	for pos := lineEnd(src, 0); pos < len(src); {
		end := lineEnd(src, pos)
		line := string(bytes.TrimRight(src[pos:end], "\r\n"))
		if slices.Contains(closers, line) {
			return end
		}
		pos = end
	}
	return 0
}

func opensWith(src []byte, delim string) bool {
	return bytes.HasPrefix(src, []byte(delim+"\n")) || bytes.HasPrefix(src, []byte(delim+"\r\n"))
}

// lineEnd returns the offset just past the end of the line starting at pos,
// including its terminator.
func lineEnd(src []byte, pos int) int {
	if i := bytes.IndexByte(src[pos:], '\n'); i >= 0 {
		return pos + i + 1
	}
	return len(src)
}
