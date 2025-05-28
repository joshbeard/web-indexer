#!/bin/bash

set -e

# Demo script for web-indexer
# Usage: ./scripts/demo.sh [local|s3|both] [--serve] [--cleanup]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_DIR="$PROJECT_ROOT/demo"
DEMO_DATA_DIR="$DEMO_DIR/data"
DEMO_OUTPUT_DIR="$DEMO_DIR/output"
DEMO_S3_BUCKET="${DEMO_S3_BUCKET:-joshbeard-web-indexer-demo-$(date +%s)}"

# Template directories
TEMPLATES_DIR="$SCRIPT_DIR/templates"
POLICY_FILE="$SCRIPT_DIR/s3-public-policy.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
DEMO_TYPE="both"
SERVE_DEMO=false
CLEANUP_ONLY=false
PERSISTENT_S3=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        local|s3|both)
            DEMO_TYPE="$1"
            shift
            ;;
        --serve)
            SERVE_DEMO=true
            shift
            ;;
        --cleanup)
            CLEANUP_ONLY=true
            shift
            ;;
        --persistent-s3)
            PERSISTENT_S3=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [local|s3|both] [--serve] [--cleanup] [--persistent-s3]"
            echo ""
            echo "Options:"
            echo "  local           Generate demo for local directories only"
            echo "  s3              Generate demo for S3 only"
            echo "  both            Generate demo for both (default)"
            echo "  --serve         Start a local web server to preview the demo"
            echo "  --cleanup       Clean up demo files and S3 resources"
            echo "  --persistent-s3 Use persistent S3 bucket (set DEMO_S3_BUCKET env var)"
            echo ""
            echo "Environment variables:"
            echo "  DEMO_S3_BUCKET  S3 bucket name (default: joshbeard-web-indexer-demo-TIMESTAMP)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

log() {
    echo -e "${GREEN}[DEMO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Template processing function
process_template() {
    local template_file="$1"
    local output_file="$2"

    if [[ ! -f "$template_file" ]]; then
        error "Template file not found: $template_file"
        return 1
    fi

    # Use sed to replace template variables
    sed \
        -e "s|{{S3_PUBLIC_URL}}|$S3_PUBLIC_URL|g" \
        -e "s|{{S3_BASE_URL}}|$S3_BASE_URL|g" \
        "$template_file" > "$output_file"
}

# Check if web-indexer binary exists
check_binary() {
    if [[ ! -f "$PROJECT_ROOT/web-indexer" ]]; then
        log "Building web-indexer binary..."
        cd "$PROJECT_ROOT"
        go build -o web-indexer .
    fi
}

# Clean up function
cleanup() {
    log "Cleaning up demo files..."

    if [[ -d "$DEMO_DIR" ]]; then
        rm -rf "$DEMO_DIR"
        log "Removed demo directory: $DEMO_DIR"
    fi

    if [[ "$DEMO_TYPE" == "s3" || "$DEMO_TYPE" == "both" ]] && [[ "$PERSISTENT_S3" == "false" ]]; then
        if command -v aws >/dev/null 2>&1; then
            log "Cleaning up S3 bucket: $DEMO_S3_BUCKET"
            # Delete all objects in the bucket
            aws s3 rm "s3://$DEMO_S3_BUCKET" --recursive 2>/dev/null || true
            # Delete the bucket
            aws s3 rb "s3://$DEMO_S3_BUCKET" 2>/dev/null || true
            log "S3 cleanup completed"
        else
            warn "AWS CLI not found, skipping S3 cleanup"
        fi
    fi
}

# Handle cleanup-only mode
if [[ "$CLEANUP_ONLY" == "true" ]]; then
    cleanup
    exit 0
fi

# Create demo data structure
create_demo_data() {
    log "Creating demo data structure..."

    mkdir -p "$DEMO_DATA_DIR"/{docs,images,downloads,projects/{project-a,project-b,project-c}}
    mkdir -p "$DEMO_OUTPUT_DIR"/{local,s3}

    # Create sample files with realistic content
    cat > "$DEMO_DATA_DIR/README.md" << 'EOF'
# Demo Repository

This is a demonstration of the web-indexer tool generating directory listings.

## Features

- Recursive directory indexing
- Multiple themes (default, solarized, nord, dracula)
- S3 and local filesystem support
- Custom templates and styling

## Contents

- `docs/` - Documentation files
- `images/` - Sample images and media
- `downloads/` - Downloadable files
- `projects/` - Sample project directories
EOF

    # Documentation files
    cat > "$DEMO_DATA_DIR/docs/installation.md" << 'EOF'
# Installation Guide

## Package Installation

Download the latest release from GitHub releases.

## Docker

```bash
docker pull joshbeard/web-indexer
```

## Building from Source

```bash
go build -o web-indexer .
```
EOF

    cat > "$DEMO_DATA_DIR/docs/configuration.md" << 'EOF'
# Configuration

Web-indexer can be configured using command-line flags or YAML configuration files.

## Basic Usage

```bash
web-indexer --source /path/to/source --target /path/to/target
```

## Configuration File

Create a `.web-indexer.yml` file in your project root.
EOF

    # Create some binary-like files (empty but with realistic names)
    touch "$DEMO_DATA_DIR/downloads/web-indexer-linux-amd64.tar.gz"
    touch "$DEMO_DATA_DIR/downloads/web-indexer-darwin-amd64.tar.gz"
    touch "$DEMO_DATA_DIR/downloads/web-indexer-windows-amd64.zip"

    # Create image placeholders
    echo "<!-- Sample image placeholder -->" > "$DEMO_DATA_DIR/images/screenshot.png"
    echo "<!-- Logo placeholder -->" > "$DEMO_DATA_DIR/images/logo.svg"

    # Project files
    cat > "$DEMO_DATA_DIR/projects/project-a/README.md" << 'EOF'
# Project A

A sample project demonstrating web-indexer capabilities.
EOF

    cat > "$DEMO_DATA_DIR/projects/project-b/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Project B</title></head>
<body><h1>Project B Custom Index</h1></body>
</html>
EOF

    # Add a .skipindex file to project-b to demonstrate the feature
    touch "$DEMO_DATA_DIR/projects/project-b/.skipindex"

    cat > "$DEMO_DATA_DIR/projects/project-c/config.yml" << 'EOF'
name: project-c
version: 1.0.0
description: Sample configuration file
EOF

    # Add a .noindex file to a subdirectory to demonstrate exclusion
    mkdir -p "$DEMO_DATA_DIR/projects/project-c/private"
    touch "$DEMO_DATA_DIR/projects/project-c/private/.noindex"
    echo "This directory should not be indexed" > "$DEMO_DATA_DIR/projects/project-c/private/secret.txt"

    log "Demo data structure created in: $DEMO_DATA_DIR"
}

# Generate local demo
generate_local_demo() {
    log "Generating local directory demo..."

    local themes=("default" "solarized" "nord" "dracula")

    for theme in "${themes[@]}"; do
        log "Generating demo with $theme theme..."
        "$PROJECT_ROOT/web-indexer" \
            --source "$DEMO_DATA_DIR" \
            --target "$DEMO_OUTPUT_DIR/local/$theme" \
            --title "Demo Repository - $theme theme" \
            --recursive \
            --theme "$theme" \
            --minify
    done

    # Create local demo index page using template
    log "Creating local demo index page..."
    cp "$TEMPLATES_DIR/local-demo-index.html" "$DEMO_OUTPUT_DIR/local/index.html"

    log "Local demo generated in: $DEMO_OUTPUT_DIR/local/"
}

# Generate S3 demo
generate_s3_demo() {
    log "Generating S3 demo..."

    # Check if AWS CLI is available
    if ! command -v aws >/dev/null 2>&1; then
        error "AWS CLI not found. Please install it to run S3 demos."
        return 1
    fi

    # Ensure AWS_REGION is set for web-indexer binary
    if [[ -z "$AWS_REGION" ]]; then
        export AWS_REGION="us-east-1"
        log "Set AWS_REGION to $AWS_REGION for web-indexer binary"
    fi

    # Create S3 bucket if it doesn't exist and we're not using persistent
    if [[ "$PERSISTENT_S3" == "false" ]]; then
        log "Creating temporary S3 bucket: $DEMO_S3_BUCKET"
        aws s3 mb "s3://$DEMO_S3_BUCKET" 2>/dev/null || {
            warn "Bucket might already exist or you might not have permissions"
        }

        # Try to disable block public access settings
        log "Configuring bucket for public access..."
        aws s3api put-public-access-block \
            --bucket "$DEMO_S3_BUCKET" \
            --public-access-block-configuration \
            "BlockPublicAcls=false,IgnorePublicAcls=false,BlockPublicPolicy=false,RestrictPublicBuckets=false" \
            2>/dev/null || warn "Could not modify public access block settings"

        # Configure bucket for static website hosting
        log "Configuring bucket for static website hosting..."
        aws s3api put-bucket-website \
            --bucket "$DEMO_S3_BUCKET" \
            --website-configuration '{
                "IndexDocument": {"Suffix": "index.html"},
                "ErrorDocument": {"Key": "error.html"}
            }' 2>/dev/null || warn "Could not configure static website hosting"

        # Try bucket policy approach
        if [[ -f "$POLICY_FILE" ]]; then
            log "Applying bucket policy for public read access..."
            local temp_policy=$(mktemp)
            sed "s/BUCKET_NAME/$DEMO_S3_BUCKET/g" "$POLICY_FILE" > "$temp_policy"

            aws s3api put-bucket-policy --bucket "$DEMO_S3_BUCKET" --policy "file://$temp_policy" 2>/dev/null || \
                warn "Could not set bucket policy (will try object-level ACLs)"

            rm -f "$temp_policy"
        else
            warn "Policy file not found: $POLICY_FILE"
        fi
    else
        log "Using persistent S3 bucket: $DEMO_S3_BUCKET"

        # Still configure website hosting for persistent buckets
        log "Configuring persistent bucket for static website hosting..."
        aws s3api put-bucket-website \
            --bucket "$DEMO_S3_BUCKET" \
            --website-configuration '{
                "IndexDocument": {"Suffix": "index.html"},
                "ErrorDocument": {"Key": "error.html"}
            }' 2>/dev/null || warn "Could not configure static website hosting"
    fi

    # Upload demo data to S3 with public-read ACL
    log "Uploading demo data to S3 with public access..."
    aws s3 sync "$DEMO_DATA_DIR" "s3://$DEMO_S3_BUCKET/data/" --delete --acl public-read 2>/dev/null || \
    aws s3 sync "$DEMO_DATA_DIR" "s3://$DEMO_S3_BUCKET/data/" --delete

    # Get the S3 region for proper URL construction
    S3_REGION=$(aws s3api get-bucket-location --bucket "$DEMO_S3_BUCKET" --query 'LocationConstraint' --output text 2>/dev/null || echo "us-east-1")
    if [[ "$S3_REGION" == "None" ]]; then
        S3_REGION="us-east-1"
    fi

    # Generate indexes for all themes from S3 data
    local themes=("default" "solarized" "nord" "dracula")

    for theme in "${themes[@]}"; do
        log "Generating S3 demo with $theme theme..."
        "$PROJECT_ROOT/web-indexer" \
            --source "s3://$DEMO_S3_BUCKET/data/" \
            --target "$DEMO_OUTPUT_DIR/s3/$theme" \
            --title "Web-Indexer Demo - $theme theme" \
            --recursive \
            --theme "$theme" \
            --minify
    done

    # Construct the public URLs (needed for template processing)
    # Use S3 website endpoints for proper index.html handling
    if [[ "$S3_REGION" == "us-east-1" ]]; then
        S3_PUBLIC_URL="http://$DEMO_S3_BUCKET.s3-website-us-east-1.amazonaws.com/"
        S3_BASE_URL="http://$DEMO_S3_BUCKET.s3-website-us-east-1.amazonaws.com"
    else
        S3_PUBLIC_URL="http://$DEMO_S3_BUCKET.s3-website-$S3_REGION.amazonaws.com/"
        S3_BASE_URL="http://$DEMO_S3_BUCKET.s3-website-$S3_REGION.amazonaws.com"
    fi

    # Create theme selector page using template
    log "Creating S3 theme selector page..."
    process_template "$TEMPLATES_DIR/s3-theme-selector.html" "$DEMO_OUTPUT_DIR/s3/index.html"

    # Create a simple error page for the website
    log "Creating error page for S3 website..."
    process_template "$TEMPLATES_DIR/s3-error.html" "$DEMO_OUTPUT_DIR/s3/error.html"

    # Upload all generated content to S3 with public access and explicit ACLs
    log "Uploading all generated S3 demo content with public access..."

    # Try with ACL first, fallback without if it fails
    if ! aws s3 sync "$DEMO_OUTPUT_DIR/s3/" "s3://$DEMO_S3_BUCKET/" --acl public-read 2>/dev/null; then
        warn "Could not set public-read ACL during sync, uploading without ACL"
        aws s3 sync "$DEMO_OUTPUT_DIR/s3/" "s3://$DEMO_S3_BUCKET/"

        # Try to set ACLs on all files recursively
        log "Attempting to set public-read ACL on all uploaded files..."

        # Get list of all uploaded objects and set ACLs
        aws s3api list-objects-v2 --bucket "$DEMO_S3_BUCKET" --query 'Contents[].Key' --output text 2>/dev/null | \
        while read -r key; do
            if [[ -n "$key" && "$key" != "None" ]]; then
                aws s3api put-object-acl --bucket "$DEMO_S3_BUCKET" --key "$key" --acl public-read 2>/dev/null || true
            fi
        done
    fi

    # Create a local redirect page for convenience
    log "Creating local redirect page..."
    process_template "$TEMPLATES_DIR/s3-redirect.html" "$DEMO_OUTPUT_DIR/s3/local-redirect.html"

    # Test the main URL
    log "Testing S3 website accessibility..."
    if curl -s -I "$S3_PUBLIC_URL" | grep -q "200 OK"; then
        log "✅ S3 website is publicly accessible!"
        log "🌐 Website endpoint: $S3_PUBLIC_URL"

        # Test a theme directory without index.html to verify website hosting
        local test_theme_url="${S3_BASE_URL}/nord/"
        if curl -s -I "$test_theme_url" | grep -q "200 OK"; then
            log "✅ Directory URLs work correctly (website hosting configured)"
        else
            warn "⚠️  Directory URLs may not work (website hosting may not be fully configured)"
        fi
    elif curl -s -I "$S3_PUBLIC_URL" | grep -q "403 Forbidden"; then
        warn "❌ S3 website is not publicly accessible (403 Forbidden)"
        warn "This may be due to:"
        warn "  - AWS account restrictions on public access"
        warn "  - Organization policies blocking public S3 access"
        warn "  - Insufficient permissions to modify bucket policies/ACLs"
        warn ""
        warn "The demo files are uploaded but may not be publicly viewable."
        warn "You can still access them if you have AWS credentials configured."
    else
        warn "⚠️  Could not determine S3 website accessibility"
    fi

    log "S3 demo with all themes generated and uploaded!"
    log "Main Demo URL: $S3_PUBLIC_URL"
    log "Theme URLs (with automatic index.html):"
    log "  Default:   $S3_BASE_URL/default/"
    log "  Solarized: $S3_BASE_URL/solarized/"
    log "  Nord:      $S3_BASE_URL/nord/"
    log "  Dracula:   $S3_BASE_URL/dracula/"

    if [[ "$PERSISTENT_S3" == "false" ]]; then
        warn "Temporary S3 bucket created. Run with --cleanup to remove it."
    fi
}

# Start local web server
serve_demo() {
    log "Starting local web server..."

    # Check what demos were generated
    local has_local_demo=false
    local has_s3_demo=false

    if [[ -d "$DEMO_OUTPUT_DIR/local" ]]; then
        has_local_demo=true
    fi

    if [[ -d "$DEMO_OUTPUT_DIR/s3" ]]; then
        has_s3_demo=true
        # For S3 demo, create a local index that redirects to the S3-hosted version
        if [[ -f "$DEMO_OUTPUT_DIR/s3/local-redirect.html" ]]; then
            cp "$DEMO_OUTPUT_DIR/s3/local-redirect.html" "$DEMO_OUTPUT_DIR/s3/index.html"
        fi
    fi

    # Create a main index page using template
    log "Creating main demo index page..."
    local temp_index=$(mktemp)
    cp "$TEMPLATES_DIR/main-demo-index.html" "$temp_index"

    # Generate demo cards based on what's available
    local local_card_file=$(mktemp)
    local s3_card_file=$(mktemp)

    if [[ "$has_local_demo" == "true" ]]; then
        cat > "$local_card_file" << 'EOF'
        <div class="demo-card">
            <h3>📁 Local Directory Demo</h3>
            <p>See how web-indexer handles local filesystem directories with multiple themes and features.</p>
            <a href="local/">Explore Local Demo →</a>
        </div>
EOF
    else
        touch "$local_card_file"
    fi

    if [[ "$has_s3_demo" == "true" ]]; then
        cat > "$s3_card_file" << 'EOF'
        <div class="demo-card">
            <h3>☁️ S3 Bucket Demo <span class="s3-badge">Live on S3</span></h3>
            <p>Discover how web-indexer can index S3 bucket contents and host the results on S3 with all themes.</p>
            <a href="s3/">Explore S3 Demo →</a>
        </div>
EOF
    else
        touch "$s3_card_file"
    fi

    # Replace placeholders in template using a more robust approach
    # First replace LOCAL_DEMO_CARD
    awk '
        /{{LOCAL_DEMO_CARD}}/ {
            while ((getline line < "'$local_card_file'") > 0) {
                print line
            }
            close("'$local_card_file'")
            next
        }
        /{{S3_DEMO_CARD}}/ {
            while ((getline line < "'$s3_card_file'") > 0) {
                print line
            }
            close("'$s3_card_file'")
            next
        }
        { print }
    ' "$temp_index" > "$DEMO_OUTPUT_DIR/index.html"

    # Clean up temporary files
    rm -f "$temp_index" "$local_card_file" "$s3_card_file"

    cd "$DEMO_OUTPUT_DIR"

    # Try different HTTP servers in order of preference
    if command -v python3 >/dev/null 2>&1; then
        info "Starting Python HTTP server on http://localhost:8080"
        python3 -m http.server 8080
    elif command -v python >/dev/null 2>&1; then
        info "Starting Python HTTP server on http://localhost:8080"
        python -m SimpleHTTPServer 8080
    elif command -v php >/dev/null 2>&1; then
        info "Starting PHP development server on http://localhost:8080"
        php -S localhost:8080
    elif command -v ruby >/dev/null 2>&1; then
        info "Starting Ruby HTTP server on http://localhost:8080"
        ruby -run -e httpd . -p 8080
    else
        warn "No suitable HTTP server found. Please install Python, PHP, or Ruby to serve the demo."
        info "You can manually serve the files from: $DEMO_OUTPUT_DIR"
        return 1
    fi
}

# Main execution
main() {
    log "Starting web-indexer demo..."
    log "Demo type: $DEMO_TYPE"

    check_binary
    create_demo_data

    case "$DEMO_TYPE" in
        "local")
            generate_local_demo
            ;;
        "s3")
            generate_s3_demo
            ;;
        "both")
            generate_local_demo
            generate_s3_demo
            ;;
    esac

    log "Demo generation completed!"
    log "Output directory: $DEMO_OUTPUT_DIR"

    if [[ "$SERVE_DEMO" == "true" ]]; then
        # Set up signal handling for cleanup only when serving
        trap cleanup EXIT
        serve_demo
    else
        info "To serve the demo locally, run: $0 --serve"
        info "Or manually serve from: $DEMO_OUTPUT_DIR"
        info "To clean up demo files, run: $0 --cleanup"
    fi
}

# Run main function
main