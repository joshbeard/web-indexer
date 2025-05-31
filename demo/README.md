# Web-Indexer Demo Tool

Generate live demonstrations of web-indexer with multiple themes and configurations.

This system is mainly intended for generating dynamic previews from pull requests.

## Quick Start

```bash
# Local demo with web server
make demo

# S3 demo (requires AWS credentials)
make demo-s3

# Both local and S3 demos
make demo-both

# Add custom demos to the standard ones
make demo CUSTOM_DEMOS="--theme nord --title 'My Custom Demo'"
make demo-s3 CUSTOM_DEMOS="custom-nord:--theme nord;minimal:--theme default --no-breadcrumbs"

# Clean up all demo files and S3 buckets
make demo-cleanup
```

## Demo Types

- **Local**: Generates demos locally and serves via HTTP server
- **S3**: Generates demos and uploads to S3 static website hosting
- **Both**: Generates both local and S3 demos

All demo types support **custom demos** - additional demos generated with custom arguments alongside the standard config-based demos.

## GitHub Integration

The demo system integrates with GitHub Actions to automatically generate live S3-hosted demos for pull requests.

### 🔒 Security & Authorization

**Demo generation requires manual approval from repository maintainers** using GitHub's [Environment Protection Rules](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-deployments/managing-environments-for-deployment).

**For contributors**:
- Use PR comments like `/demo` or `/demo --args "custom args"`
- A maintainer will review and approve your demo request
- Fork PRs automatically require approval for security

### 🚀 Usage

**Triggers** (authorized users only):
- PR comments: `/demo`, `/demo --args "custom args"`
- PR labels: Add/remove `demo` label
- Manual: GitHub Actions → Preview workflow

**Process**: Builds from PR → Generates demos → Deploys to S3 → Comments with URLs → Auto-cleanup on PR close

### 📋 Available Commands

| Command | Description |
|---------|-------------|
| `/demo` | Generate all theme demos |
| `/demo --args "custom args"` | Generate themes + custom demo |
| `/demo cleanup` | Clean up S3 resources |
| `/build` | Generate dev Docker image |
| `/build cleanup` | Clean up Docker images |

## Configuration

Demo configuration is in [`config.yml`](config.yml). This defines:

- Demo metadata (title, description)
- S3 settings (bucket prefix, region)
- Server settings (port)
- Demo specifications (themes, arguments, directories)

## Custom Demos

You can generate additional demos with custom arguments alongside the standard config-based demos:

### Local Usage

```bash
# Add a single custom demo
make demo CUSTOM_DEMOS="--theme nord --title 'My Custom Demo'"

# Add multiple custom demos (semicolon-separated)
make demo CUSTOM_DEMOS="nord-demo:--theme nord --title 'Nordic Theme';minimal:--theme default --no-breadcrumbs"

# For S3 demos
make demo-s3 CUSTOM_DEMOS="--theme dracula --minify --sort-by last_modified"

# Direct usage with demo.go
go run demo.go -custom-demos "test-demo:--theme nord --title 'Test Demo'" -serve
```

### CI Usage (GitHub Pull Requests)

See the **[GitHub Integration](#github-integration)** section above for comprehensive details on using demos in GitHub PRs.

Quick examples:
- `/demo --args "--theme nord --title 'My Custom Demo'"`
- `/demo --args "nord-demo:--theme nord;minimal:--theme default"`

### Custom Demo Format

- Simple: `"--theme nord --title 'Demo'"` (auto-named as `custom-1`, `custom-2`, etc.)
- Named: `"my-demo:--theme nord --title 'Demo'"` (creates demo named `my-demo`)
- Multiple: `"demo1:--args1;demo2:--args2"` (semicolon-separated)

### Custom Configuration

```bash
# Use your own config file
make demo-custom CONFIG=my-config.yml
```

To create a custom config:
1. Copy `demo/config.yml` to `my-config.yml`
2. Modify settings (especially `s3.bucket_prefix`)
3. Add/remove demos as needed

## Environment Variables

For S3 demos:
- `AWS_ACCESS_KEY_ID` - AWS access key (required)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key (required)
- `AWS_REGION` - AWS region (default: from config)
- `DEMO_S3_BUCKET` - Override bucket name (optional)

## Files

- `demo.go` - Demo generation script
- `config.yml` - Demo configuration
- `Makefile` - Demo-specific make targets
- `templates/` - Demo content templates
- `data/` - Generated demo data (git-ignored)
- `output/` - Generated demo output (git-ignored)
- `.demo-buckets.json` - S3 bucket tracking (git-ignored)

## Running Demos

Demo targets can be run from either the project root or the `demo/` directory:

```bash
# From project root
make demo-s3

# From demo directory
cd demo && make demo-s3
```

Both approaches work identically. The demo system has its own `Makefile` in the `demo/` directory.

## Cleanup

```bash
# Clean up demo files and tracked S3 buckets
make demo-cleanup

# Emergency cleanup of ALL S3 buckets with your prefix
make demo-cleanup-s3-all
```

S3 buckets are automatically tracked in `.demo-buckets.json` and cleaned up when you run `make demo-cleanup`.

## Make Targets

| Target | Description |
|--------|-------------|
| `demo` | Generate and serve local demo |
| `demo-local` | Generate local demo only |
| `demo-s3` | Generate S3 demo only |
| `demo-both` | Generate both local and S3 demos |
| `demo-custom CONFIG=file` | Use custom configuration file |
| `demo-with-custom CUSTOM="args"` | Generate demos with custom arguments |
| `demo-cleanup` | Clean up demo files and tracked S3 buckets |
| `demo-cleanup-s3-all` | Emergency cleanup of all prefixed S3 buckets |

**Note**: All `demo*` targets support `CUSTOM_DEMOS="args"` parameter to add custom demos.