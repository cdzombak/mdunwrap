# my-project

A small utility that does one thing well, written in Go. This description is
hard-wrapped at roughly eighty columns, which is exactly the kind of wrapping
this tool exists to undo.

## Installation

Install with Homebrew:

```shell
brew install example/tap/my-project
```

Or build from source. A working Go installation is required, and the build
embeds a version string derived from git:

```shell
git clone https://github.com/example/my-project.git
cd my-project
go build -o /usr/local/bin/my-project .
```

## Usage

The tool reads from standard input and writes to standard output by default,
so it composes with other command line tools in the usual way.

```text
my-project [options] [file]
```

### Options

- `-verbose`: Print additional detail about what the tool is doing, which is
  useful when diagnosing unexpected output.
- `-quiet`: Suppress all non-error output. This is the inverse of `-verbose`
  and the two may not be combined.
- `-format`: Choose the output format. The supported values are `text`, which
  is the default, and `json`.

## Configuration

Configuration is read from a file whose location follows the XDG base
directory specification, falling back to the user's home directory when the
relevant environment variables are unset.

| Key       | Default | Description                          |
|-----------|---------|--------------------------------------|
| `timeout` | `30s`   | How long to wait before giving up    |
| `retries` | `3`     | Number of times to retry on failure  |

> [!NOTE]
> The configuration file is optional. When it is absent the defaults shown
> above are used, and no warning is emitted.

## Caveats

1. The tool assumes its input is UTF-8 encoded, and will report an error
   rather than guess at another encoding.
2. Very large inputs are buffered entirely in memory, which is fine for the
   documents this tool is meant to process.

## License

Apache 2.0; see LICENSE in this repo.
