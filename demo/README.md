# Web-Indexer Demo Tool

Generate live demonstrations of web-indexer with multiple configurations.

## Quick Start

```bash
# Local demo with web server
make demo

# S3 demo (requires AWS credentials)
make demo-s3

# Both local and S3 demos
make demo-both

# Add custom demos
make demo CUSTOM_DEMOS="--theme nord --title 'My Custom Demo'"

# Clean up all demo files and S3 buckets
make demo-cleanup
```

## Demo Types

- **Local**: Generates demos locally and serves via HTTP server
- **S3**: Generates demos and uploads to S3 static website hosting
- **Both**: Generates both local and S3 demos

## GitHub Integration

The demo system automatically generates live previews for pull requests and releases.

### Pull Request Previews

**Security**: Demo generation requires manual approval from repository maintainers using GitHub's Environment Protection Rules.

**Commands** (use in PR comments):
- `/preview` - Generate all configured theme demos
- `/preview --args "custom args"` - Generate themes + custom demo with specified args
- `/preview cleanup` - Clean up S3 resources for this PR

**Preview URLs**: `https://web-indexer.jbeard.dev/pr/{PR_NUMBER}/`

### Release Previews

Release previews are automatically generated when commits are pushed to the main branch and deployed to a persistent URL.

## Configuration

Demo configuration is in [`config.yml`](config.yml). The configuration defines:

- S3 settings (bucket prefix, region)
- Local server port
- List of demos with their themes and arguments

## Custom Demos

Add custom demos alongside the configured ones:

```bash
# Single custom demo
make demo CUSTOM_DEMOS="--theme nord --title 'My Demo'"

# Multiple custom demos (semicolon-separated)
make demo CUSTOM_DEMOS="nord:--theme nord;minimal:--theme default --no-breadcrumbs"

# Named format: "name:args" or simple format: "args" (auto-named)
```

### GitHub PR Usage

```
/preview --args "--theme dracula --title 'Dark Theme Test'"
/preview --args "nord:--theme nord;minimal:--theme default"
```

## Environment Variables

**For S3 demos:**
- `AWS_ACCESS_KEY_ID` - AWS access key (required)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key (required)
- `AWS_REGION` - AWS region (default: from config)
- `DEMO_S3_BUCKET` - Override bucket name (optional)

## Files

- `demo.go` - Demo generation script
- `config.yml` - Demo configuration
- `Makefile` - Demo make targets
- `templates/` - Demo content templates
- `data/` - Generated demo data (git-ignored)
- `output/` - Generated demo output (git-ignored)
- `.demo-buckets.json` - S3 bucket tracking (git-ignored)

## Make Targets

| Target | Description |
|--------|-------------|
| `demo` | Generate and serve local demo |
| `demo-local` | Generate local demo only |
| `demo-s3` | Generate S3 demo only |
| `demo-both` | Generate both local and S3 demos |
| `demo-cleanup` | Clean up demo files and tracked S3 buckets |

**Note**: All targets support `CUSTOM_DEMOS="args"` parameter.

## Cleanup

```bash
# Clean up demo files and tracked S3 buckets
make demo-cleanup

# Cleanup of ALL S3 buckets with your prefix
make demo-cleanup-s3-all
```

S3 buckets are automatically tracked in `.demo-buckets.json` and cleaned up when you run `make demo-cleanup`.

PR preview environments are automatically cleaned up when PRs are closed.