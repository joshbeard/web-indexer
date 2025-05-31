package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration file
type Config struct {
	Demo   DemoSettings   `yaml:"demo"`
	S3     S3Settings     `yaml:"s3"`
	Server ServerSettings `yaml:"server"`
	Demos  []DemoSpec     `yaml:"demos"`
}

type DemoSettings struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type S3Settings struct {
	BucketPrefix string `yaml:"bucket_prefix"`
	Region       string `yaml:"region"`
}

type ServerSettings struct {
	Port int `yaml:"port"`
}

type DemoSpec struct {
	Name        string `yaml:"name"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Args        string `yaml:"args"`
	Directory   string `yaml:"directory"`
}

type DemoConfig struct {
	Type             string `json:"type"`               // local, s3, both
	Serve            bool   `json:"serve"`              // start web server
	Cleanup          bool   `json:"cleanup"`            // cleanup only
	ConfigFile       string `json:"config_file"`        // path to YAML config file
	ProjectRoot      string `json:"project_root"`       // project root directory
	DemoDataDir      string `json:"demo_data_dir"`      // demo data directory
	DemoOutputDir    string `json:"demo_output_dir"`    // demo output directory
	WebIndexerBinary string `json:"web_indexer_binary"` // path to web-indexer binary
	TemplatesDir     string `json:"templates_dir"`      // templates directory
	S3Bucket         string `json:"s3_bucket"`          // S3 bucket name
	S3Region         string `json:"s3_region"`          // S3 region
	S3PublicURL      string `json:"s3_public_url"`      // S3 public URL
	CustomDemos      string `json:"custom_demos"`       // semicolon-separated custom demo specs

	// Configuration loaded from YAML
	Config *Config `json:"-"`
}

type DemoIndex struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Demos       []DemoSpec `json:"demos"`
}

type BucketRecord struct {
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	Created   time.Time `json:"created"`
	ConfigDir string    `json:"config_dir"`
}

func main() {
	demoType := flag.String("type", "local", "Demo type: local, s3, both")
	serve := flag.Bool("serve", false, "Start web server to preview demo")
	cleanup := flag.Bool("cleanup", false, "Clean up demo files")
	configFile := flag.String("config", "config.yml", "Path to YAML configuration file")
	customDemos := flag.String("custom-demos", "", "Semicolon-separated custom demo specs (format: 'name:args' or just 'args')")
	help := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	config := &DemoConfig{
		Type:        *demoType,
		Serve:       *serve,
		Cleanup:     *cleanup,
		ConfigFile:  *configFile,
		CustomDemos: *customDemos,
	}

	if err := loadConfig(config); err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if err := setupPaths(config); err != nil {
		log.Fatalf("Error setting up paths: %v", err)
	}

	if config.Cleanup {
		if err := cleanupDemo(config); err != nil {
			log.Fatalf("Error cleaning up: %v", err)
		}
		return
	}

	logf("Starting web-indexer demo...")
	logf("Demo type: %s", config.Type)
	logf("Demo variant: Configured demos from %s", config.ConfigFile)

	if err := runDemo(config); err != nil {
		log.Fatalf("Error running demo: %v", err)
	}

	if config.Serve {
		if err := serveDemo(config); err != nil {
			log.Fatalf("Error serving demo: %v", err)
		}
	}
}

func showHelp() {
	fmt.Printf(`Web-Indexer Demo Generator

Usage: %s [options]

Options:
  -type string
        Demo type: local, s3, both (default "local")
  -serve
        Start web server to preview demo
  -cleanup
        Clean up demo files
  -config string
        Path to YAML configuration file (default "config.yml")
  -custom-demos string
        Semicolon-separated custom demo specs (format: 'name:args' or just 'args')
  -help
        Show this help

Examples:
  %s -serve
  %s -type s3
  %s -type both -serve
  %s -config my-config.yml -serve
  %s -cleanup

  # Add custom demos in addition to config-based ones
  %s -custom-demos "--theme nord --title 'Custom Demo'"
  %s -custom-demos "custom-nord:--theme nord --title 'Custom Demo';minimal:--theme default --no-breadcrumbs"

Environment Variables (for S3 demos):
  DEMO_S3_BUCKET         S3 bucket name (default: auto-generated with timestamp)
  AWS_REGION             AWS region (default: us-east-1)
  AWS_ACCESS_KEY_ID      AWS access key
  AWS_SECRET_ACCESS_KEY  AWS secret key

Cleanup Tips:
  # Clean up demo files and ALL tracked S3 buckets
  %s -cleanup

  # Clean up ALL S3 buckets with your prefix (including untracked ones)
  make demo-cleanup-s3-all

  # Bucket tracking: S3 buckets are automatically tracked in demo/.demo-buckets.json
  # and cleaned up when you run -cleanup

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

// loadConfig loads the YAML configuration file
func loadConfig(demoConfig *DemoConfig) error {
	if _, err := os.Stat(demoConfig.ConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("config file %s not found. Please ensure the configuration file exists", demoConfig.ConfigFile)
	}

	data, err := os.ReadFile(demoConfig.ConfigFile)
	if err != nil {
		return fmt.Errorf("reading config file %s: %w", demoConfig.ConfigFile, err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parsing YAML config: %w", err)
	}

	if err := validateDemos(config); err != nil {
		return fmt.Errorf("validating demos: %w", err)
	}

	// Parse and add custom demos if provided
	if demoConfig.CustomDemos != "" {
		customDemos, err := parseCustomDemos(demoConfig.CustomDemos)
		if err != nil {
			return fmt.Errorf("parsing custom demos: %w", err)
		}

		if len(customDemos) > 0 {
			logf("Adding %d custom demo(s) to %d config-based demo(s)", len(customDemos), len(config.Demos))
			config.Demos = append(config.Demos, customDemos...)
		}
	}

	demoConfig.Config = config
	logf("Loaded configuration from %s", demoConfig.ConfigFile)
	if demoConfig.CustomDemos != "" {
		logf("Added custom demos: %s", demoConfig.CustomDemos)
	}
	return nil
}

func setupPaths(config *DemoConfig) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	config.ProjectRoot = wd
	config.DemoDataDir = filepath.Join(wd, "data")
	config.DemoOutputDir = filepath.Join(wd, "output")
	config.WebIndexerBinary = filepath.Join(wd, "..", "web-indexer")
	config.TemplatesDir = filepath.Join(wd, "templates")

	if config.Type == "s3" || config.Type == "both" {
		return setupS3Config(config)
	}
	return nil
}

func setupS3Config(config *DemoConfig) error {
	if bucket := os.Getenv("DEMO_S3_BUCKET"); bucket != "" {
		config.S3Bucket = bucket
	} else {
		config.S3Bucket = fmt.Sprintf("%s-%d", config.Config.S3.BucketPrefix, time.Now().Unix())
	}

	// Validate S3 bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	if region := os.Getenv("AWS_REGION"); region != "" {
		config.S3Region = region
	} else {
		config.S3Region = config.Config.S3.Region
		if err := os.Setenv("AWS_REGION", config.S3Region); err != nil {
			return fmt.Errorf("setting AWS_REGION environment variable: %w", err)
		}
	}

	if config.S3Region == "us-east-1" {
		config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-us-east-1.amazonaws.com/", config.S3Bucket)
	} else {
		config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-%s.amazonaws.com/", config.S3Bucket, config.S3Region)
	}

	testCmd := exec.Command("aws", "sts", "get-caller-identity")
	testCmd.Stderr = nil
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("AWS credentials not available. Please run 'aws configure' or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")
	}

	logf("S3 Configuration:")
	logf("  Bucket: %s", config.S3Bucket)
	logf("  Region: %s", config.S3Region)
	logf("  Public URL: %s", config.S3PublicURL)

	return nil
}

func logf(format string, args ...interface{}) {
	fmt.Printf("[DEMO] "+format+"\n", args...)
}

func runDemo(config *DemoConfig) error {
	if err := checkBinary(config); err != nil {
		return err
	}

	if err := createDemoData(config); err != nil {
		return err
	}

	switch config.Type {
	case "local":
		return generateLocalDemo(config)
	case "s3":
		return generateS3Demo(config)
	case "both":
		if err := generateLocalDemo(config); err != nil {
			return fmt.Errorf("generating local demo: %w", err)
		}
		return generateS3Demo(config)
	default:
		return fmt.Errorf("unknown demo type: %s", config.Type)
	}
}

func checkBinary(config *DemoConfig) error {
	if _, err := os.Stat(config.WebIndexerBinary); os.IsNotExist(err) {
		logf("Building web-indexer binary...")
		cmd := exec.Command("go", "build", "-o", "web-indexer", ".")
		cmd.Dir = config.ProjectRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("building web-indexer: %w", err)
		}
	}
	return nil
}

func createDemoData(config *DemoConfig) error {
	logf("Creating demo data structure...")

	dirs, err := discoverDirectoriesFromTemplates(config)
	if err != nil {
		return fmt.Errorf("discovering directories from templates: %w", err)
	}

	baseDirs := []string{
		config.DemoDataDir,
		filepath.Join(config.DemoOutputDir, "local"),
	}

	if config.Type == "s3" || config.Type == "both" {
		baseDirs = append(baseDirs, filepath.Join(config.DemoOutputDir, "s3"))
	}

	dirs = append(dirs, baseDirs...)

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	templateContentDir := filepath.Join(config.TemplatesDir, "demo-content")
	if err := copyTemplateFiles(templateContentDir, config.DemoDataDir); err != nil {
		return fmt.Errorf("copying template files: %w", err)
	}

	logf("Demo data structure created in: %s", config.DemoDataDir)
	return nil
}

func discoverDirectoriesFromTemplates(config *DemoConfig) ([]string, error) {
	templateContentDir := filepath.Join(config.TemplatesDir, "demo-content")
	var dirs []string

	err := filepath.Walk(templateContentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			relPath, err := filepath.Rel(templateContentDir, path)
			if err != nil {
				return err
			}

			if relPath != "." {
				dirs = append(dirs, filepath.Join(config.DemoDataDir, relPath))
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking template directory %s: %w", templateContentDir, err)
	}

	logf("Discovered %d directories from templates", len(dirs))
	return dirs, nil
}

// copyTemplateFiles recursively copies files from source to destination
func copyTemplateFiles(src, dst string) error {
	// Validate source and destination paths
	if err := validateFilePath(src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validateFilePath(dst); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Validate the current path being processed
		if err := validateFilePath(path); err != nil {
			return fmt.Errorf("invalid file path %s: %w", path, err)
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		// Validate destination path
		if err := validateFilePath(dstPath); err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// #nosec G304 -- path is validated by validateFilePath() above
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// #nosec G304 -- dstPath is validated by validateFilePath() above
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = srcFile.WriteTo(dstFile)
		return err
	})
}

func generateLocalDemo(config *DemoConfig) error {
	logf("Generating local directory demo...")
	return generateThemesLocalDemo(config)
}

func generateThemesLocalDemo(config *DemoConfig) error {
	logf("Generating local demo with all configured demos...")

	for _, demo := range config.Config.Demos {
		logf("Generating demo: %s", demo.Name)
		targetDir := filepath.Join(config.DemoOutputDir, "local", demo.Directory)

		args := []string{
			"--source", config.DemoDataDir,
			"--target", targetDir,
		}

		if demo.Args != "" {
			customArgs, err := parseArgs(demo.Args)
			if err != nil {
				return fmt.Errorf("parsing demo args for %s: %w", demo.Name, err)
			}
			args = append(args, customArgs...)
		}

		if err := runWebIndexer(config, args); err != nil {
			return fmt.Errorf("generating %s demo: %w", demo.Name, err)
		}

		if err := copySourceFilesToTarget(config.DemoDataDir, targetDir); err != nil {
			logf("Warning: Could not copy source files to %s: %v", targetDir, err)
		}
	}

	logf("Creating local demo index page...")
	indexData := DemoIndex{
		Title:       fmt.Sprintf("%s - Local", config.Config.Demo.Title),
		Description: fmt.Sprintf("%s. This demo shows web-indexer generating directory listings with different themes from local filesystem data.", config.Config.Demo.Description),
		Demos:       config.Config.Demos,
	}

	return generateIndexPage(config, indexData, "local")
}

func generateIndexPage(config *DemoConfig, data DemoIndex, variant string) error {
	outputPath := filepath.Join(config.DemoOutputDir, variant, "index.html")
	templatePath := filepath.Join(config.TemplatesDir, "demo-index.html")

	// Validate paths
	if err := validateFilePath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	if err := validateFilePath(templatePath); err != nil {
		return fmt.Errorf("invalid template path: %w", err)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parsing template file %s: %w", templatePath, err)
	}

	// #nosec G304 -- outputPath is validated by validateFilePath() above
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating index file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing index file: %w", err)
	}

	return nil
}

func cleanupDemo(config *DemoConfig) error {
	logf("Cleaning up demo files...")

	dirsToClean := []string{
		config.DemoDataDir,
		config.DemoOutputDir,
	}

	for _, dir := range dirsToClean {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			logf("Warning: Could not remove %s: %v", dir, err)
		} else if err == nil {
			logf("Cleaned up: %s", dir)
		}
	}

	if err := cleanupS3Bucket(config); err != nil {
		logf("Warning: Error cleaning up S3 buckets: %v", err)
	}

	logf("Cleanup completed")
	return nil
}

func cleanupS3Bucket(config *DemoConfig) error {
	logf("Cleaning up tracked S3 buckets...")

	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found, skipping S3 cleanup")
	}

	buckets, err := getTrackedBuckets(config)
	if err != nil {
		return fmt.Errorf("getting tracked buckets: %w", err)
	}

	if len(buckets) == 0 {
		logf("No tracked S3 buckets found")
		return nil
	}

	logf("Found %d tracked bucket(s):", len(buckets))
	for _, bucket := range buckets {
		logf("  - %s (created %s)", bucket.Name, bucket.Created.Format("2006-01-02 15:04:05"))
	}

	for _, bucket := range buckets {
		logf("Cleaning up bucket: %s", bucket.Name)

		// Validate bucket name
		if err := validateS3BucketName(bucket.Name); err != nil {
			logf("Warning: Invalid bucket name %s, skipping: %v", bucket.Name, err)
			continue
		}

		// #nosec G204 -- bucket.Name is validated by validateS3BucketName() above
		deleteCmd := exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s", bucket.Name), "--recursive")
		if err := deleteCmd.Run(); err != nil {
			logf("Warning: Could not delete objects in bucket %s: %v", bucket.Name, err)
		}

		// #nosec G204 -- bucket.Name is validated by validateS3BucketName() above
		rbCmd := exec.Command("aws", "s3", "rb", fmt.Sprintf("s3://%s", bucket.Name))
		if err := rbCmd.Run(); err != nil {
			logf("Warning: Could not delete bucket %s: %v", bucket.Name, err)
		} else {
			logf("Successfully deleted bucket: %s", bucket.Name)
			if err := removeTrackedBucket(config, bucket.Name); err != nil {
				logf("Warning: Could not remove bucket from tracking: %v", err)
			}
		}
	}

	logf("S3 cleanup completed")
	return nil
}

func serveDemo(config *DemoConfig) error {
	port := config.Config.Server.Port
	dir := filepath.Join(config.DemoOutputDir, "local")

	logf("👉 Local Demo URL: http://localhost:%d", port)
	logf("   Serving from: %s", dir)
	logf("-----------------------------------------------------------------------")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      http.FileServer(http.Dir(dir)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// generateS3Demo generates demos for S3 with S3 as both source and target
func generateS3Demo(config *DemoConfig) error {
	logf("Generating S3 demo with S3 as source and target...")

	// Create bucket if it doesn't exist
	if err := createS3BucketIfNotExists(config); err != nil {
		return fmt.Errorf("creating S3 bucket: %w", err)
	}

	// Track the bucket for later cleanup
	if err := trackS3Bucket(config); err != nil {
		logf("Warning: Could not track S3 bucket: %v", err)
	}

	// Upload sample data to S3 first
	if err := uploadSampleDataToS3(config); err != nil {
		return fmt.Errorf("uploading sample data to S3: %w", err)
	}

	// Generate demos using S3 as both source and target
	if err := generateS3DemosWithS3Source(config); err != nil {
		return fmt.Errorf("generating S3 demos with S3 source: %w", err)
	}

	// Generate main demo index page locally and upload
	if err := generateAndUploadS3IndexPage(config); err != nil {
		return fmt.Errorf("generating S3 index page: %w", err)
	}

	// Create and upload S3 error page
	if err := createAndUploadS3ErrorPage(config); err != nil {
		logf("Warning: Could not create S3 error page: %v", err)
	}

	logf("S3 demo with all configured demos generated!")
	logf("-----------------------------------------------------------------------")
	logf("👉 S3 Demo URL: %s", config.S3PublicURL)

	return nil
}

func uploadSampleDataToS3(config *DemoConfig) error {
	logf("Uploading sample data to S3 data directory...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Ensure demo data exists locally first
	if err := createDemoData(config); err != nil {
		return fmt.Errorf("creating demo data: %w", err)
	}

	// Upload sample data to s3://bucket/data/
	dataS3Path := fmt.Sprintf("s3://%s/data/", config.S3Bucket)
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	syncCmd := exec.Command("aws", "s3", "sync", config.DemoDataDir, dataS3Path)
	output, err := syncCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uploading sample data to S3: %w\nOutput: %s", err, string(output))
	}

	logf("Successfully uploaded sample data to %s", dataS3Path)
	return nil
}

func generateS3DemosWithS3Source(config *DemoConfig) error {
	logf("Generating S3 demos using S3 as source and target...")

	dataS3URI := fmt.Sprintf("s3://%s/data", config.S3Bucket)

	for _, demo := range config.Config.Demos {
		logf("Generating S3 demo: %s", demo.Name)

		targetS3URI := fmt.Sprintf("s3://%s/%s", config.S3Bucket, demo.Directory)

		args := []string{
			"--source", dataS3URI,
			"--target", targetS3URI,
		}

		// Parse and add arguments from config
		if demo.Args != "" {
			customArgs, err := parseArgs(demo.Args)
			if err != nil {
				return fmt.Errorf("parsing demo args for %s: %w", demo.Name, err)
			}
			args = append(args, customArgs...)
		}

		logf("Running web-indexer with S3 source and target: %s → %s", dataS3URI, targetS3URI)
		if err := runWebIndexer(config, args); err != nil {
			return fmt.Errorf("generating %s demo with S3 source: %w", demo.Name, err)
		}
	}

	return nil
}

func generateAndUploadS3IndexPage(config *DemoConfig) error {
	logf("Generating and uploading main S3 demo index page...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Create S3 demo index page locally
	indexData := DemoIndex{
		Title:       fmt.Sprintf("%s - S3", config.Config.Demo.Title),
		Description: fmt.Sprintf("%s. This demo shows web-indexer with S3 as both source and target.", config.Config.Demo.Description),
		Demos:       config.Config.Demos,
	}

	// Generate locally in temp directory
	tempDir := filepath.Join(config.DemoOutputDir, "temp-s3-index")
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate index page
	outputPath := filepath.Join(tempDir, "index.html")
	templatePath := filepath.Join(config.TemplatesDir, "demo-index.html")

	// Validate paths
	if err := validateFilePath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	if err := validateFilePath(templatePath); err != nil {
		return fmt.Errorf("invalid template path: %w", err)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parsing template file %s: %w", templatePath, err)
	}

	// #nosec G304 -- outputPath is validated by validateFilePath() above
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating index file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, indexData); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing index file: %w", err)
	}

	// Upload to S3 root
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	uploadCmd := exec.Command("aws", "s3", "cp", outputPath, fmt.Sprintf("s3://%s/index.html", config.S3Bucket))
	output, err := uploadCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uploading index page to S3: %w\nOutput: %s", err, string(output))
	}

	logf("Successfully uploaded main index page to S3")
	return nil
}

func createS3BucketIfNotExists(config *DemoConfig) error {
	// Validate bucket name first
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Check if AWS CLI is available
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found")
	}

	// Check if the bucket already exists
	logf("Checking if S3 bucket exists: %s", config.S3Bucket)
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	existsCmd := exec.Command("aws", "s3", "ls", fmt.Sprintf("s3://%s", config.S3Bucket))
	if err := existsCmd.Run(); err == nil {
		logf("S3 bucket already exists")
		return nil
	}

	// Create the bucket
	logf("Creating S3 bucket: %s", config.S3Bucket)
	var createCmd *exec.Cmd
	if config.S3Region == "us-east-1" {
		// For us-east-1, don't specify region
		// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
		createCmd = exec.Command("aws", "s3", "mb", fmt.Sprintf("s3://%s", config.S3Bucket))
	} else {
		// For other regions, specify the region
		// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
		createCmd = exec.Command("aws", "s3", "mb", fmt.Sprintf("s3://%s", config.S3Bucket), "--region", config.S3Region)
	}

	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating S3 bucket: %w\nOutput: %s", err, string(output))
	}

	// Configure static website hosting
	if err := configureS3StaticWebsiteHosting(config); err != nil {
		logf("Warning: Could not configure static website hosting: %v", err)
	}

	logf("S3 bucket created successfully")
	return nil
}

func configureS3StaticWebsiteHosting(config *DemoConfig) error {
	logf("Configuring S3 static website hosting...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Check if AWS CLI is available
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found, skipping static website hosting configuration")
	}

	// Configure static website hosting
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	configureCmd := exec.Command("aws", "s3", "website", fmt.Sprintf("s3://%s", config.S3Bucket), "--index-document", "index.html", "--error-document", "error.html")
	if err := configureCmd.Run(); err != nil {
		return fmt.Errorf("configuring S3 static website hosting: %w", err)
	}

	// Disable block public access
	if err := disableS3BlockPublicAccess(config); err != nil {
		logf("Warning: Could not disable block public access: %v", err)
	}

	// Set bucket policy for public read access
	if err := setS3BucketPolicy(config); err != nil {
		logf("Warning: Could not set bucket policy for public access: %v", err)
	}

	logf("S3 static website hosting configured successfully")
	return nil
}

func disableS3BlockPublicAccess(config *DemoConfig) error {
	logf("Disabling S3 block public access...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Check if AWS CLI is available
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found, skipping block public access configuration")
	}

	// Disable block public access
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	disableCmd := exec.Command("aws", "s3api", "delete-public-access-block", "--bucket", config.S3Bucket)
	output, err := disableCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disabling S3 block public access: %w\nOutput: %s", err, string(output))
	}

	logf("S3 block public access disabled successfully")
	return nil
}

func setS3BucketPolicy(config *DemoConfig) error {
	logf("Setting S3 bucket policy for public read access...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	// Create bucket policy JSON
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "PublicReadGetObject",
				"Effect": "Allow",
				"Principal": "*",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`, config.S3Bucket)

	// Write policy to temporary file
	tempFile, err := os.CreateTemp("", "s3-policy-*.json")
	if err != nil {
		return fmt.Errorf("creating temporary policy file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := tempFile.WriteString(policy); err != nil {
		return fmt.Errorf("writing policy to temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Apply bucket policy
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName(), tempFile.Name() is controlled
	policyCmd := exec.Command("aws", "s3api", "put-bucket-policy", "--bucket", config.S3Bucket, "--policy", fmt.Sprintf("file://%s", tempFile.Name()))
	output, err := policyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("setting bucket policy: %w\nOutput: %s", err, string(output))
	}

	logf("S3 bucket policy set successfully")
	return nil
}

func createAndUploadS3ErrorPage(config *DemoConfig) error {
	logf("Creating and uploading S3 error page...")

	// Validate bucket name
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	errorTemplatePath := filepath.Join(config.TemplatesDir, "s3-error.html")
	localErrorPath := filepath.Join(config.DemoOutputDir, "s3-error.html")

	// Validate paths
	if err := validateFilePath(errorTemplatePath); err != nil {
		return fmt.Errorf("invalid error template path: %w", err)
	}
	if err := validateFilePath(localErrorPath); err != nil {
		return fmt.Errorf("invalid local error path: %w", err)
	}

	// Read the error template
	// #nosec G304 -- errorTemplatePath is validated by validateFilePath() above
	errorData, err := os.ReadFile(errorTemplatePath)
	if err != nil {
		return fmt.Errorf("reading error template %s: %w", errorTemplatePath, err)
	}

	// Write error page locally
	if err := os.WriteFile(localErrorPath, errorData, 0o600); err != nil {
		return fmt.Errorf("writing local error page %s: %w", localErrorPath, err)
	}

	// Upload to S3
	// #nosec G204 -- config.S3Bucket is validated by validateS3BucketName()
	uploadCmd := exec.Command("aws", "s3", "cp", localErrorPath, fmt.Sprintf("s3://%s/error.html", config.S3Bucket))
	output, err := uploadCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uploading error page to S3: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// validateDemos validates that all demos have required fields
func validateDemos(config *Config) error {
	for i, demo := range config.Demos {
		if demo.Name == "" {
			return fmt.Errorf("demo at index %d is missing name", i)
		}
		if demo.Args == "" {
			return fmt.Errorf("demo %s is missing args", demo.Name)
		}
		if demo.Directory == "" {
			return fmt.Errorf("demo %s is missing directory", demo.Name)
		}
		if demo.Title == "" {
			config.Demos[i].Title = demo.Name
		}
	}
	return nil
}

func runWebIndexer(config *DemoConfig, args []string) error {
	// Validate binary path
	if err := validateFilePath(config.WebIndexerBinary); err != nil {
		return fmt.Errorf("invalid web-indexer binary path: %w", err)
	}

	// Ensure the binary exists and is executable
	if _, err := os.Stat(config.WebIndexerBinary); err != nil {
		return fmt.Errorf("web-indexer binary not found: %w", err)
	}

	// #nosec G204 -- WebIndexerBinary path is validated and constructed from trusted config
	cmd := exec.Command(config.WebIndexerBinary, args...)
	cmd.Dir = config.ProjectRoot

	logf("Running command: '%s %s' in directory: %s", config.WebIndexerBinary, strings.Join(args, " "), config.ProjectRoot)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running web-indexer with args %v: %w\nOutput: %s", args, err, string(output))
	}

	return nil
}

// parseArgs parses a command line string into arguments, preserving quoted strings
func parseArgs(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	var inQuote bool
	var quoteChar byte

	for i := 0; i < len(s); i++ {
		c := s[i]

		switch {
		case !inQuote && (c == '"' || c == '\''):
			inQuote = true
			quoteChar = c
		case inQuote && c == quoteChar:
			inQuote = false
		case !inQuote && c == ' ':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}

	if inQuote {
		return nil, fmt.Errorf("unclosed quote in arguments: %s", s)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}

// copySourceFilesToTarget copies source files to target directory
func copySourceFilesToTarget(sourceDir, targetDir string) error {
	// Validate source and target directories
	if err := validateFilePath(sourceDir); err != nil {
		return fmt.Errorf("invalid source directory: %w", err)
	}
	if err := validateFilePath(targetDir); err != nil {
		return fmt.Errorf("invalid target directory: %w", err)
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Validate the current path being processed
		if err := validateFilePath(path); err != nil {
			return fmt.Errorf("invalid file path %s: %w", path, err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(targetDir, relPath)

		// Validate destination path
		if err := validateFilePath(dstPath); err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// #nosec G304 -- path is validated by validateFilePath() above
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// #nosec G304 -- dstPath is validated by validateFilePath() above
		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = srcFile.WriteTo(dstFile)
		return err
	})
}

func trackS3Bucket(config *DemoConfig) error {
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")

	// Validate file path
	if err := validateFilePath(bucketFile); err != nil {
		return fmt.Errorf("invalid bucket file path: %w", err)
	}

	var buckets []BucketRecord
	// #nosec G304 -- bucketFile is validated by validateFilePath() above
	if data, err := os.ReadFile(bucketFile); err == nil {
		if err := json.Unmarshal(data, &buckets); err != nil {
			return fmt.Errorf("unmarshaling bucket records: %w", err)
		}
	}

	record := BucketRecord{
		Name:      config.S3Bucket,
		Region:    config.S3Region,
		Created:   time.Now(),
		ConfigDir: filepath.Dir(config.ConfigFile),
	}

	exists := false
	for i, bucket := range buckets {
		if bucket.Name == config.S3Bucket {
			buckets[i] = record
			exists = true
			break
		}
	}

	if !exists {
		buckets = append(buckets, record)
	}

	data, err := json.MarshalIndent(buckets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bucket records: %w", err)
	}

	if err := os.WriteFile(bucketFile, data, 0o600); err != nil {
		return fmt.Errorf("writing bucket records: %w", err)
	}

	logf("Recorded bucket in %s", bucketFile)
	return nil
}

func getTrackedBuckets(config *DemoConfig) ([]BucketRecord, error) {
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")

	// Validate file path
	if err := validateFilePath(bucketFile); err != nil {
		return nil, fmt.Errorf("invalid bucket file path: %w", err)
	}

	var buckets []BucketRecord
	// #nosec G304 -- bucketFile is validated by validateFilePath() above
	data, err := os.ReadFile(bucketFile)
	if err != nil {
		if os.IsNotExist(err) {
			return buckets, nil
		}
		return nil, fmt.Errorf("reading bucket records: %w", err)
	}

	if err := json.Unmarshal(data, &buckets); err != nil {
		return nil, fmt.Errorf("parsing bucket records: %w", err)
	}

	return buckets, nil
}

func removeTrackedBucket(config *DemoConfig, bucketName string) error {
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")

	// Validate file path and bucket name
	if err := validateFilePath(bucketFile); err != nil {
		return fmt.Errorf("invalid bucket file path: %w", err)
	}
	if err := validateS3BucketName(bucketName); err != nil {
		return fmt.Errorf("invalid bucket name: %w", err)
	}

	buckets, err := getTrackedBuckets(config)
	if err != nil {
		return err
	}

	var filteredBuckets []BucketRecord
	for _, bucket := range buckets {
		if bucket.Name != bucketName {
			filteredBuckets = append(filteredBuckets, bucket)
		}
	}

	data, err := json.MarshalIndent(filteredBuckets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bucket records: %w", err)
	}

	return os.WriteFile(bucketFile, data, 0o600)
}

// parseCustomDemos parses semicolon-separated custom demo specifications
func parseCustomDemos(customDemosStr string) ([]DemoSpec, error) {
	if customDemosStr == "" {
		return nil, nil
	}

	var customDemos []DemoSpec
	specs := strings.Split(customDemosStr, ";")

	for i, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}

		var name, args string

		// Check if spec contains name:args format
		if colonIndex := strings.Index(spec, ":"); colonIndex > 0 {
			name = strings.TrimSpace(spec[:colonIndex])
			args = strings.TrimSpace(spec[colonIndex+1:])
		} else {
			// Auto-generate name if not provided
			name = fmt.Sprintf("custom-%d", i+1)
			args = spec
		}

		if args == "" {
			return nil, fmt.Errorf("empty args for custom demo: %s", spec)
		}

		// Sanitize name for directory usage
		directory := strings.ReplaceAll(strings.ToLower(name), " ", "-")
		directory = strings.ReplaceAll(directory, ":", "-")

		customDemo := DemoSpec{
			Name:        name,
			Title:       fmt.Sprintf("Custom: %s", name),
			Description: fmt.Sprintf("Custom demo generated with: %s", args),
			Args:        args,
			Directory:   directory,
		}

		customDemos = append(customDemos, customDemo)
	}

	return customDemos, nil
}

// validateS3BucketName validates S3 bucket name to prevent injection attacks
func validateS3BucketName(bucket string) error {
	if bucket == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	// S3 bucket name validation (simplified)
	// Must be 3-63 characters, lowercase letters, numbers, dots, hyphens
	matched, err := regexp.MatchString(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`, bucket)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid S3 bucket name: %s", bucket)
	}

	if len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("S3 bucket name must be 3-63 characters")
	}

	return nil
}

// validateFilePath validates file paths to prevent directory traversal
func validateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Resolve the path to its absolute form and clean it
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: contains directory traversal")
	}

	// Check for other dangerous patterns
	if strings.Contains(cleanPath, "~") {
		return fmt.Errorf("invalid path: contains tilde expansion")
	}

	// Check for null bytes (common in path injection attacks)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid path: contains null bytes")
	}

	// Additional check: ensure the path doesn't start with certain dangerous patterns
	if strings.HasPrefix(cleanPath, "/proc/") || strings.HasPrefix(cleanPath, "/sys/") {
		return fmt.Errorf("invalid path: access to system directories not allowed")
	}

	return nil
}
