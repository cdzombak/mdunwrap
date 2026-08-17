---
title: Nothing here is joinable
tags:
  - one
  - two
---

# Every line in this file must survive untouched

## A heading, then a rule

---

* One item
* Another item
* A third item

1. First
2. Second
3. Third

> A single-line quote.

> Another quote.

| Column | Other |
|--------|-------|
| a      | b     |
| c      | d     |

```go
func main() {
	fmt.Println("this looks like wrapped prose but is code")
	fmt.Println("second line")
}
```

~~~
tilde fenced block
second line
~~~

    indented code block
    second line

<div class="note">
  <p>An HTML block.</p>
</div>

<!--
A multi-line HTML comment.
-->

A paragraph with a hard break at the end.  
The next line after the hard break.

Another paragraph ending in a backslash break.\
And its continuation.

Text containing `a code  
span that wraps` and must not change.

[ref-one]: https://example.com/one
[ref-two]: https://example.com/two

:::note
:::

$$
E = mc^2
$$

Setext Heading
==============

Another Setext
--------------

A final single-line paragraph.
