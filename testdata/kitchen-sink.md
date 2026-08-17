---
title: A gnarly document
description: frontmatter values
  may wrap and must not
  be touched
---

# A heading that is
wrapped in the source

Wait, no: the line above starts a new paragraph, because an ATX heading is a
single line. This paragraph itself is hard-wrapped at a narrow width and
should collapse onto one line.

A paragraph with an explicit hard break at the end of this line  
and a continuation that must stay on its own line, but this part
of the same paragraph may still join.

A backslash break appears here\
and again the continuation stays put.

## Lists

- A list item whose text is wrapped
  across two source lines
- A second item that is also
  wrapped onto another line
- [ ] An unchecked task that
  wraps as well
- [x] A checked task
  wrapping too

1. An ordered item that
   wraps here
2. Another ordered item
   that wraps

- An item with a nested list
  - A nested item that
    wraps at depth
  - Another nested item

- An item containing a fence

  ```python
  def f():
      # not prose
      return 1
  ```

## Blockquotes

> A blockquote whose prose is
> hard-wrapped across lines
> and keeps going.
>
> - A quoted list item that
>   wraps inside the quote

> A quote with a lazy continuation
that has no marker at all.

> > A nested quote that is
> > wrapped as well.

## Tables

| Name | Description |
|------|-------------|
| a    | first row   |
| b    | second row  |

Some prose directly above a table:

Name | Age
-----|----
Bob  | 42
Eve  | 37

## Code

```go
func main() {
	// These comment lines look like wrapped
	// prose but they are code and must not move.
}
```

    An indented code block
    with a second line

Text with `a code  
span crossing lines` that must not be joined.

## HTML

<div class="wrapper">
  <p>An HTML block whose
  lines must not be joined.</p>
</div>

<!-- A comment
spanning lines -->

## Extensions

:::note
An admonition body that is
wrapped and should join.
:::

$$
x = 1
y = 2
$$

Term one
: A definition that is
  wrapped across lines

[^1]: A footnote definition
    whose body wraps

A paragraph referencing the footnote[^1] and wrapped
across two lines.

Setext Heading That Is
Wrapped In Source
======================

[link-ref]: https://example.com/very/long/url
