# `mdunwrap`

Remove unnecessary line breaks from Markdown, for readers whose viewer wraps text itself.

If you prefer your Markdown viewer to soft-wrap paragraphs, hard-wrapped source is just noise. This tool joins the line breaks that exist only to wrap prose to a column width, and leaves every line break that carries meaning.

## What it does and doesn't touch

Joined:

- Wrapped paragraph text
- Wrapped list item text, keeping the marker and indentation
- Wrapped text inside blockquotes, keeping the `>` prefix
- Wrapped setext heading text, footnote bodies, and definition descriptions

Left exactly as they were:

- Headings, thematic breaks, and blank lines
- Separate list items, and nested lists
- Table rows, including the header and delimiter rows
- Fenced and indented code blocks, and multi-line code spans
- HTML blocks and comments
- YAML and TOML frontmatter
- Explicit hard breaks: a line ending in two spaces or a backslash
- `$$` math blocks, `:::note` admonitions, and `> [!NOTE]` alerts

Blocks are located with [goldmark](https://github.com/yuin/goldmark), a CommonMark-compliant parser, so structure is recognized properly rather than guessed at with regular expressions. The result is then rendered and compared against the input, and any join that would have changed the rendered output is discarded. Everything outside the joins is copied through byte for byte, and running the tool twice produces the same result as running it once.

## Usage

```text
mdunwrap [options] [file]
```

Either an input (`-in`, or a bare filename) or `-in-place` is required.

### Arguments

- `-in`: Path to the input file, or `-` for stdin.
- `-out`: Path to the output file, or `-` for stdout. Defaults to stdout.
- `-in-place`: Path to a file which is read, used as input, and replaced by the output. Cannot be combined with `-in` or `-out`.
- `-force`: Allow `-out` to overwrite an existing file. Without it, writing to a path that already exists is an error.
- `-version`: Print version and exit.
- `-help`: Print help and exit.

A single bare filename argument is equivalent to `-in`.

`-in-place` writes to a temporary file in the same directory and renames it over the original, so an interrupted run cannot leave a truncated document behind. The file's permissions are preserved.

## Usage Examples

Unwrap a file to stdout:

```text
$ mdunwrap notes.md
# Notes

This paragraph was hard-wrapped in the source and is now a single line.
```

Rewrite a file in place:

```text
$ mdunwrap -in-place notes.md
```

Read from a pipe:

```text
$ pbpaste | mdunwrap -in - | pbcopy
```

Write to a new file, refusing to clobber it:

```text
$ mdunwrap -in notes.md -out wrapped.md
$ mdunwrap -in notes.md -out wrapped.md
wrapped.md already exists; pass -force to overwrite it
$ mdunwrap -in notes.md -out wrapped.md -force
```

## Installation

### macOS via Homebrew

```shell
brew install cdzombak/oss/mdunwrap
```

### Debian/Ubuntu and derivatives, via apt repository

Install my Debian repository if you haven't already:

```shell
sudo apt-get install ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/deb.key | sudo gpg --dearmor -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 0644 /etc/apt/keyrings/dist-cdzombak-net.gpg
echo -e "deb [signed-by=/etc/apt/keyrings/dist-cdzombak-net.gpg] https://dist.cdzombak.net/deb/oss any oss\n" | sudo tee -a /etc/apt/sources.list.d/dist-cdzombak-net.list > /dev/null
sudo apt-get update
```

Then install `mdunwrap` via `apt-get`:

```shell
sudo apt-get install mdunwrap
```

### From source

A working Go installation is required.

```shell
git clone https://github.com/cdzombak/mdunwrap.git
cd mdunwrap
go build -ldflags="-X main.version=$(./.version.sh)" -o /usr/local/bin/mdunwrap .
```

## License

Apache 2.0; see LICENSE in this repo.

## Author

Chris Dzombak
- [dzombak.com](https://www.dzombak.com)
- [github.com/cdzombak](https://github.com/cdzombak)
