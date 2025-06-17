# web-indexer

Quick and simple program to generate basic directory index pages from a local
file directory or S3 bucket, such as a file listing page.

<p align="center">
  <img width="420" height="236" alt="screenshot" src=".github/readme/screenshot.png" />
</p>
<p align="center">
  <a href="https://web-indexer.jbeard.dev">Demo</a>
</p>

This isn't a good solution for a dynamic listing of directories or S3 buckets,
but it's simple and works for content that remains static between
deployments. This is particularly helpful for hosting static artifacts on S3,
since S3 doesn't natively offer content indexes.

## Alternatives

If you're using Nginx, [fancy_index](https://www.nginx.com/resources/wiki/modules/fancy_index/)
is the right tool for the job and does this dynamically with full customization.

If you're using Apache, [mod_autoindex](https://httpd.apache.org/docs/2.4/mod/mod_autoindex.html) is
what you're looking for, maybe with something like [Apaxy](https://oupala.github.io/apaxy/).

Options for S3 include [flightlesstux/S3-Directory-Listing](https://github.com/flightlesstux/S3-Directory-Listing)
and [rufuspollock/s3-bucket-listing](https://github.com/rufuspollock/s3-bucket-listing).

## Usage

There's several ways to use web-indexer:

- **Package:** Download packages from the [releases](https://github.com/joshbeard/web-indexer/releases)
- **Docker:** Use the Docker image from [`ghcr.io/joshbeard/web-indexer/web-indexer`](https://github.com/joshbeard/web-indexer/pkgs/container/web-indexer%2Fweb-indexer)
  or [`joshbeard/web-indexer`](https://hub.docker.com/r/joshbeard/web-indexer) on the Docker Hub
- **CI/CD:** Use the [GitHub Action](#github-action) or a [GitLab job](#gitlab-ci).

```plain
Usage:
  web-indexer --source <source> --target <target> [flags]

Flags:
  -u, --base-url string         A URL to prepend to the links
  -c, --config string           config file
      --date-format string      The date format to use in the index page (default "2006-01-02 15:04:05 MST")
      --dirs-first              List directories first (default true)
  -h, --help                    help for web-indexer
  -i, --index-file string       The name of the index file (default "index.html")
  -l, --link-to-index           Link to the index file or just the path
      --link-up-from-root       Show a parent/up link even when at the root of the indexed path
      --link-up-text string     Text to display for the up link from root (default "Go Up")
      --link-up-url string      URL path for the up link from root (default "..")
  -F, --log-file string         The log file
  -L, --log-level string        The log level (default "info")
  -m, --minify                  Minify the index page
  -n, --noindex-files strings   A list of files that indicate a directory should be skipped. Comma separated or specified multiple times (default [.noindex])
      --order string            The order for the items. One of: asc, desc (default "asc")
  -q, --quiet                   Suppress log output
  -r, --recursive               List files recursively
  -S, --skip strings            A list of files or directories to skip. Comma separated or specified multiple times
      --skipindex-files strings A list of files that indicate a directory should be skipped for indexing but still included in the parent directory listing. Comma separated or specified multiple times (default [.skipindex])
      --sort-by string          The order for the index page. One of: last_modified, name, natural_name (default "natural_name")
  -s, --source string           REQUIRED. The source directory or S3 URI to list
  -t, --target string           REQUIRED. The target directory or S3 URI to write to
  -f, --template string         A custom template file to use for the index page
      --theme string            The theme to use for the index page. One of: default, solarized, nord, dracula (default "default")
  -T, --title string            The title of the index page
  -v, --version                 version for web-indexer
```

### Examples

The `source` and `target` arguments can be specified using their respective
flags, or provided as two unflagged arguments:

```shell
web-indexer SOURCE TARGET
web-indexer --source SOURCE --target TARGET
```

Index a local directory and write the index file to the same directory:

```shell
web-indexer --source /path/to/directory --target /path/to/directory
```

Index a local directory and write the index file to a different directory:

```shell
web-indexer --source /path/to/directory --target /foo/bar
```

Index a local directory and upload the index file to an S3 bucket:

```shell
web-indexer --source /path/to/directory --target s3://bucket/path
```

Index an S3 bucket and write the index file to a local directory:

```shell
web-indexer --source s3://bucket/path --target /path/to/directory
```

Index an S3 bucket and upload the index file to the same bucket and path:

```shell
web-indexer --source s3://bucket/path --target s3://bucket/path
```

Set a title for the index pages:

```shell
web-indexer --source /path/to/directory --target /path/to/directory --title 'Index of {relativePath}'
```

Use a specific theme:

```shell
web-indexer --source /path/to/directory --target /path/to/directory --theme solarized
```

Enable a parent/up link from the root directory:

```shell
web-indexer --source /path/to/directory --target /path/to/directory --link-up-from-root --link-up-text "Back to Parent" --link-up-url "../"
```

Load a config:

```shell
web-indexer -config /path/to/config.yaml
```

## Themes

Web-indexer comes with several built-in themes that provide different visual styles for your directory listings. All themes are responsive and include both light and dark mode variants that automatically adapt to the user's system preferences.

### Available Themes

- **default**: The original theme with a clean, minimal design supporting both light and dark modes dynamically
- **solarized**: Based on the popular [Solarized](https://ethanschoonover.com/solarized/) color scheme with carefully chosen colors for optimal readability
- **nord**: Inspired by the [Nord](https://www.nordtheme.com/) color palette with its Arctic-inspired colors
- **dracula**: Based on the popular [Dracula](https://draculatheme.com/) dark theme with vibrant, high-contrast colors

### Using Themes

You can select a theme using the `--theme` flag:

```shell
web-indexer --source /path/to/directory --target /path/to/directory --theme nord
```

Or in your configuration file:

```yaml
source: /path/to/directory
target: /path/to/directory
theme: dracula
```

### Custom Templates

If the built-in themes don't meet your needs, you can still use a completely custom template with the `--template` flag:

```shell
web-indexer --source /path/to/directory --target /path/to/directory --template /path/to/custom/template.html
```

## GitHub Action

web-indexer is also available as a GitHub action.

For example:

```yaml
jobs:
  build:
    steps:
      # ... Other config ...
      - name: S3 Index Generator
        uses: joshbeard/web-indexer@0.4.1
        with:
          config: .web-indexer.yml
          # Optional: Use a specific image tag (e.g., for PR testing)
          # image_tag: dev-pr123
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: 'us-east-1'
```

## GitLab CI

```yaml
# Ensure you're authorized with AWS when using S3
web-index:
  image:
    name: joshbeard/web-indexer
    entrypoint: ['']
  script:
    - web-indexer
      --source dist/web/
      --target s3://my-bucket/public/
      --title "Project Files"
      --recursive
      --theme nord
```

## Configuration

You can configure the behavior of `web-indexer` using command-line arguments
and/or a YAML config file. Both are evaluated with the command-line arguments
taking precedence.

Configuration files named `.web-indexer.yml` or `.web-indexer.yaml` in the
current working directory will be automatically loaded.

The full configuration with default values for each key are provided below:

```yaml
# base_url is an optional URL to prefix to links. If unset, links are relative.
base_url: ""

# date_format is the date format to use for indexed files modified time.
# This is provided in Go's `time` package format.
# See https://pkg.go.dev/time#pkg-examples
date_format: "2006-01-02 15:04:05 UTC"

# dirs_first toggles if directories should be ordered before files in the
# list.
dirs_first: true

# index_file is the name of the file to generate.
index_file: "index.html"

# link_to_index toggles linking to the index_file for sub-paths or just the
# root of the subpath (foo/ vs foo/index.html).
link_to_index: false

# link_up_from_root enables showing a parent/up link even when at the root
# of the indexed path.
link_up_from_root: false

# link_up_text is the text to display for the up link from root.
link_up_text: "Go Up"

# link_up_url is the URL path for the up link from root.
link_up_url: ".."

# log_file is an optional path to a file to log to
# if log_level is set and this is not, messages are logged to stderr
log_file: ""

# log_level configures the verbosity of logging.
# Acceptable values: info, error, warn, debug
# Set this to an empty string or use the 'quiet' option  to suppress all log
# output.
log_level: "info"

# minify toggles minifying the generated HTML.
minify: false

# noindex_files is a list of filenames that, when present in a directory,
# indicate that the directory and its subdirectories should be skipped.
noindex_files: [".noindex"]

# quiet suppresses all log output
quiet: false

# order the items (asc)ending or (desc)ending (by sort).
order: "asc"

# recursive enables indexing the source recursively.
recursive: false

# skipindex_files is a list of filenames that, when present in a directory,
# indicate that the directory should be skipped for indexing but still
# included in the parent directory's listing.
skipindex_files: [".skipindex"]

# skips is a list of filenames to skip.
skips: []

# sort_by determines how the items are sorted.
# Valid values: last_modified, name, name_natural
# name_natural sorts by name in a human friendly way (e.g. 1,2,10 not 1,10,2).
sort_by: "name_natural"

# source is the path to a local directory or an S3 URI.
source: "blah/"

# target is the path to a local directory or an S3 URI.
target: "blah/"

# template is the path to a local Go template file to use for generating the
# indexes. The built-in template is used by default.
template: ""

# theme is the name of the built-in theme to use.
# Valid values: default, solarized, nord, dracula
theme: "default"

# title customizes the title field available in the template.
# Certain tokens can be used to be dynamically replaced.
#   {source}       - the base source path
#   {target}       - the target path or URI
#   {path}         - the full path including the source
#   {relativePath} - the path relative to the source
title: ""
```

### Example Configuration

```yaml
source: /mnt/repos
target: s3://my-cool-repos/
title: "Index of {relativePath}"
minify: true
recursive: true
link_to_index: true
link_up_from_root: true
link_up_text: "Back to Repository Root"
link_up_url: "../"
order: desc
theme: nord
```

## Excluding Directories with .noindex Files

You can exclude directories from being indexed by placing a `.noindex` file (or any file specified with the `--noindex-files` flag) in those directories. When the indexer encounters a directory containing a noindex file, it will:

1. Skip generating an index for that directory
2. Skip including that directory in the parent directory's index
3. Skip all subdirectories beneath it

This is useful for:
- Excluding private or sensitive directories from being indexed
- Preventing indexing of directories that contain temporary or build files
- Selectively controlling which directories appear in your indexes

## Excluding Directories from Indexing but Including in Parent Listing

You can also use the `--skipindex-files` flag to specify files that, when present in a directory, indicate that directory should be excluded from indexing but still appear in the parent directory's listing:

1. Skip generating an index for that directory (same as `--noindex-files`)
2. **Still include the directory in the parent directory's index** (different from `--noindex-files`)
3. Skip all subdirectories beneath it (same as `--noindex-files`)

This is particularly useful for directories that already have their own index pages (e.g., generated by a web framework) where you want to link to them from the parent directory, but don't want to overwrite their custom index.

### Examples

To exclude a directory using the default `.noindex` file:

```shell
# Create an empty .noindex file in a directory you want to exclude completely
touch /path/to/directory/.noindex
```

To exclude a directory from indexing but still include it in the parent's index:

```shell
# Create an empty .skipindex file
touch /path/to/directory/.skipindex
```

To specify custom noindex filenames:

```shell
# Use custom filenames instead of the default .noindex
web-indexer --source /path/to/directory --target /path/to/output --noindex-files .private,.hidden,DO_NOT_INDEX
```

To specify custom skipindex filenames:

```shell
# Use custom filenames instead of the default .skipindex
web-indexer --source /path/to/directory --target /path/to/output --skipindex-files index.html,.custom-index
```

In your YAML configuration:

```yaml
source: /path/to/directory
target: /path/to/output
noindex_files:
  - .private
  - .hidden
  - DO_NOT_INDEX
skipindex_files:
  - index.html
  - .custom-index
```

When the indexer encounters a directory with a noindex file, it will log a message at the INFO level:

```
INFO Skipping /path/to/directory (found noindex file .noindex)
```

When the indexer encounters a directory with a skipindex file, it will log a message at the INFO level:

```
INFO Skipping indexing of /path/to/directory (found skipindex file .skipindex), will include in parent directory
```
