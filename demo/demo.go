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

type Config struct {
	S3     S3Settings     `yaml:"s3"`
	Server ServerSettings `yaml:"server"`
	Demos  []DemoSpec     `yaml:"demos"`
}

type S3Settings struct {
	BucketPrefix string `yaml:"bucket_prefix"`
	Region       string `yaml:"region"`
	CustomDomain string `yaml:"custom_domain"`
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
	PRNumber    string     `json:"pr_number,omitempty"`
	PRUrl       string     `json:"pr_url,omitempty"`
	CustomArgs  string     `json:"custom_args,omitempty"`
	Repository  string     `json:"repository,omitempty"`
}

type BucketRecord struct {
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	Created   time.Time `json:"created"`
	ConfigDir string    `json:"config_dir"`
}

func main() {
	demoType := flag.String("type", "local", "Demo type: local, s3, both, s3-release")
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
        Demo type: local, s3, both, s3-release (default "local")
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
  %s -custom-demos "--theme nord --title 'Custom Demo'"
  %s -cleanup

Environment Variables (for S3 demos):
  DEMO_S3_BUCKET         S3 bucket name (default: auto-generated)
  AWS_REGION             AWS region (default: us-east-1)
  AWS_ACCESS_KEY_ID      AWS access key
  AWS_SECRET_ACCESS_KEY  AWS secret key

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func loadConfig(demoConfig *DemoConfig) error {
	if _, err := os.Stat(demoConfig.ConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("config file %s not found", demoConfig.ConfigFile)
	}

	data, err := os.ReadFile(demoConfig.ConfigFile)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parsing YAML config: %w", err)
	}

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

	if demoConfig.CustomDemos != "" {
		customDemos, err := parseCustomDemos(demoConfig.CustomDemos)
		if err != nil {
			return fmt.Errorf("parsing custom demos: %w", err)
		}
		config.Demos = append(customDemos, config.Demos...)
		logf("Added %d custom demo(s)", len(customDemos))
	}

	demoConfig.Config = config
	logf("Loaded %d demo(s) from configuration", len(config.Demos))
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

	if strings.Contains(config.Type, "s3") {
		return setupS3Config(config)
	}
	return nil
}

func setupS3Config(config *DemoConfig) error {
	if bucket := os.Getenv("DEMO_S3_BUCKET"); bucket != "" {
		config.S3Bucket = bucket
	} else if config.Type == "s3-release" {
		return fmt.Errorf("DEMO_S3_BUCKET environment variable is required for release preview")
	} else {
		config.S3Bucket = fmt.Sprintf("%s-%d", config.Config.S3.BucketPrefix, time.Now().Unix())
	}

	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return fmt.Errorf("invalid S3 bucket name: %w", err)
	}

	if region := os.Getenv("AWS_REGION"); region != "" {
		config.S3Region = region
	} else {
		config.S3Region = config.Config.S3.Region
		os.Setenv("AWS_REGION", config.S3Region)
	}
	if shouldUseCustomDomain(config) {
		config.S3PublicURL = buildCustomDomainURL(config)
	} else {
		if config.S3Region == "us-east-1" {
			config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-us-east-1.amazonaws.com/", config.S3Bucket)
		} else {
			config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-%s.amazonaws.com/", config.S3Bucket, config.S3Region)
		}
	}

	if err := exec.Command("aws", "sts", "get-caller-identity").Run(); err != nil {
		return fmt.Errorf("AWS credentials not available. Please configure AWS CLI")
	}

	logf("S3 Config - Bucket: %s, Region: %s", config.S3Bucket, config.S3Region)
	return nil
}

func shouldUseCustomDomain(config *DemoConfig) bool {
	return config.Config.S3.CustomDomain != "" && os.Getenv("GITHUB_ACTIONS") == "true"
}

func buildCustomDomainURL(config *DemoConfig) string {
	baseURL := strings.TrimSuffix(config.Config.S3.CustomDomain, "/")

	previewPath := strings.TrimSuffix(baseURL, "/")

	if config.Type == "s3-release" {
		return fmt.Sprintf("%s/", baseURL)
	}

	if prNumber := os.Getenv("PR_NUMBER"); prNumber != "" {
		return fmt.Sprintf("%s%s/%s/", baseURL, previewPath, prNumber)
	}

	return fmt.Sprintf("%s%s/", baseURL, previewPath)
}

func runDemo(config *DemoConfig) error {
	if err := checkBinary(config); err != nil {
		return err
	}

	if err := createDemoData(config); err != nil {
		return err
	}

	if err := generateSourceTree(config); err != nil {
		logf("Warning: Could not generate source tree: %v", err)
	}

	switch config.Type {
	case "local":
		return generateLocalDemo(config)
	case "s3":
		return generateS3Demo(config)
	case "s3-release":
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

	dirs := []string{
		config.DemoDataDir,
		filepath.Join(config.DemoOutputDir, "local"),
	}
	if strings.Contains(config.Type, "s3") {
		dirs = append(dirs, filepath.Join(config.DemoOutputDir, "s3"))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	templateContentDir := filepath.Join(config.TemplatesDir, "demo-content")
	if err := copyFiles(templateContentDir, config.DemoDataDir); err != nil {
		return fmt.Errorf("copying template files: %w", err)
	}

	return nil
}

func generateLocalDemo(config *DemoConfig) error {
	logf("Generating local demos...")

	for _, demo := range config.Config.Demos {
		if err := generateSingleDemo(config, demo, "local"); err != nil {
			return fmt.Errorf("generating demo %s: %w", demo.Name, err)
		}
	}

	return generateIndexPage(config, "local")
}

func generateS3Demo(config *DemoConfig) error {
	logf("Generating S3 demos...")

	if err := createS3BucketIfNotExists(config); err != nil {
		return fmt.Errorf("setting up S3 bucket: %w", err)
	}

	if err := trackS3Bucket(config); err != nil {
		logf("Warning: Could not track S3 bucket: %v", err)
	}

	if err := uploadSampleDataToS3(config); err != nil {
		return fmt.Errorf("uploading sample data: %w", err)
	}

	if err := uploadErrorPageToS3(config); err != nil {
		return fmt.Errorf("uploading error page: %w", err)
	}

	// Generate demos using S3 as source and target
	dataS3URI := fmt.Sprintf("s3://%s/data", config.S3Bucket)
	for _, demo := range config.Config.Demos {
		targetS3URI := fmt.Sprintf("s3://%s/%s", config.S3Bucket, demo.Directory)
		if err := generateS3SingleDemo(config, demo, dataS3URI, targetS3URI); err != nil {
			return fmt.Errorf("generating S3 demo %s: %w", demo.Name, err)
		}
	}

	// Source tree is now part of the source data and gets synced automatically

	// Clean up the temporary data directory
	logf("Cleaning up temporary data directory...")
	exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s/data", config.S3Bucket), "--recursive").Run()

	if err := generateAndUploadS3IndexPage(config); err != nil {
		return fmt.Errorf("generating S3 index page: %w", err)
	}

	logf("S3 demo complete! URL: %s", config.S3PublicURL)
	return nil
}

func generateSingleDemo(config *DemoConfig, demo DemoSpec, demoType string) error {
	logf("Generating %s demo: %s", demoType, demo.Name)

	var sourceDir, targetDir string
	if demoType == "local" {
		sourceDir = config.DemoDataDir
		targetDir = filepath.Join(config.DemoOutputDir, "local", demo.Directory)
	}

	args := []string{"--source", sourceDir, "--target", targetDir}
	if demo.Args != "" {
		customArgs, err := parseArgs(demo.Args)
		if err != nil {
			return fmt.Errorf("parsing args: %w", err)
		}
		args = append(args, customArgs...)
	}

	if err := runWebIndexer(config, args); err != nil {
		return fmt.Errorf("running web-indexer: %w", err)
	}

	// Copy source files to target for local demos
	if demoType == "local" {
		if err := copyFiles(sourceDir, targetDir); err != nil {
			logf("Warning: Could not copy source files: %v", err)
		}
	}

	return nil
}

func generateS3SingleDemo(config *DemoConfig, demo DemoSpec, sourceS3URI, targetS3URI string) error {
	logf("Generating S3 demo: %s (%s -> %s)", demo.Name, sourceS3URI, targetS3URI)

	args := []string{"--source", sourceS3URI, "--target", targetS3URI}
	if demo.Args != "" {
		customArgs, err := parseArgs(demo.Args)
		if err != nil {
			return fmt.Errorf("parsing args: %w", err)
		}
		args = append(args, customArgs...)
	}

	if err := runWebIndexer(config, args); err != nil {
		return fmt.Errorf("running web-indexer: %w", err)
	}

	// Copy source files to target for S3 demos
	logf("Copying source files from %s to %s", sourceS3URI, targetS3URI)
	syncCmd := exec.Command("aws", "s3", "sync", sourceS3URI+"/", targetS3URI+"/")
	if err := syncCmd.Run(); err != nil {
		logf("Warning: Could not copy S3 source files: %v", err)
	}

	return nil
}

func generateIndexPage(config *DemoConfig, variant string) error {
	logf("Generating %s index page...", variant)

	prNumber := os.Getenv("PR_NUMBER")
	repository := os.Getenv("GITHUB_REPOSITORY")
	customArgs := os.Getenv("CUSTOM_ARGS")
	releaseVersion := os.Getenv("RELEASE_VERSION")

	var title string
	if config.Type == "s3-release" && releaseVersion != "" {
		title = fmt.Sprintf("Web-Indexer Release Preview - v%s", releaseVersion)
	} else if prNumber != "" {
		title = fmt.Sprintf("Web-Indexer Preview - PR #%s", prNumber)
	} else {
		title = "Web-Indexer Preview"
	}

	indexData := DemoIndex{
		Title:       title,
		Description: "Web-indexer generates themeable directory listings",
		Demos:       config.Config.Demos,
		PRNumber:    prNumber,
		Repository:  repository,
		CustomArgs:  customArgs,
	}

	if repository != "" && prNumber != "" {
		indexData.PRUrl = fmt.Sprintf("https://github.com/%s/pull/%s", repository, prNumber)
	}

	outputPath := filepath.Join(config.DemoOutputDir, variant, "index.html")
	templatePath := filepath.Join(config.TemplatesDir, "index.html")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating index file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, indexData)
}

func generateAndUploadS3IndexPage(config *DemoConfig) error {
	tempDir := filepath.Join(config.DemoOutputDir, "temp-s3-index")
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	originalOutputDir := config.DemoOutputDir
	config.DemoOutputDir = tempDir

	if err := generateIndexPage(config, "."); err != nil {
		config.DemoOutputDir = originalOutputDir
		return err
	}
	config.DemoOutputDir = originalOutputDir
	indexPath := filepath.Join(tempDir, "index.html")
	uploadCmd := exec.Command("aws", "s3", "cp", indexPath, fmt.Sprintf("s3://%s/index.html", config.S3Bucket))
	if err := uploadCmd.Run(); err != nil {
		return fmt.Errorf("uploading index to S3: %w", err)
	}

	return nil
}

func cleanupDemo(config *DemoConfig) error {
	logf("Cleaning up demo files...")

	dirs := []string{config.DemoDataDir, config.DemoOutputDir}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			logf("Warning: Could not remove %s: %v", dir, err)
		}
	}
	if err := cleanupS3Bucket(config); err != nil {
		logf("Warning: S3 cleanup error: %v", err)
	}

	logf("Cleanup completed")
	return nil
}

func cleanupS3Bucket(config *DemoConfig) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return nil
	}

	// If DEMO_S3_BUCKET is set, clean up that specific bucket
	if specificBucket := os.Getenv("DEMO_S3_BUCKET"); specificBucket != "" {
		logf("Cleaning up specific S3 bucket: %s", specificBucket)
		if err := validateS3BucketName(specificBucket); err != nil {
			logf("Warning: Invalid bucket name %s: %v", specificBucket, err)
		} else {
			exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s", specificBucket), "--recursive").Run()
			if err := exec.Command("aws", "s3", "rb", fmt.Sprintf("s3://%s", specificBucket)).Run(); err == nil {
				logf("Deleted bucket: %s", specificBucket)
				removeTrackedBucket(config, specificBucket)
			} else {
				logf("Warning: Could not delete bucket %s (may not exist)", specificBucket)
				// Still try to remove from tracking in case it's stale
				removeTrackedBucket(config, specificBucket)
			}
		}
		return nil
	}

	// Otherwise, clean up all tracked buckets
	buckets, err := getTrackedBuckets(config)
	if err != nil || len(buckets) == 0 {
		return err
	}

	logf("Cleaning up %d tracked S3 bucket(s)", len(buckets))
	for _, bucket := range buckets {
		if err := validateS3BucketName(bucket.Name); err != nil {
			continue
		}

		exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s", bucket.Name), "--recursive").Run()
		if err := exec.Command("aws", "s3", "rb", fmt.Sprintf("s3://%s", bucket.Name)).Run(); err == nil {
			logf("Deleted bucket: %s", bucket.Name)
			removeTrackedBucket(config, bucket.Name)
		}
	}

	return nil
}

func serveDemo(config *DemoConfig) error {
	port := config.Config.Server.Port
	dir := filepath.Join(config.DemoOutputDir, "local")

	logf("👉 Local Demo URL: http://localhost:%d", port)
	logf("   Serving from: %s", dir)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      http.FileServer(http.Dir(dir)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func logf(format string, args ...interface{}) {
	fmt.Printf("[DEMO] "+format+"\n", args...)
}

func runWebIndexer(config *DemoConfig, args []string) error {
	cmd := exec.Command(config.WebIndexerBinary, args...)
	cmd.Dir = config.ProjectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("web-indexer failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

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

func copyFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = srcFile.WriteTo(dstFile)
		return err
	})
}

func parseCustomDemos(customDemosStr string) ([]DemoSpec, error) {
	if customDemosStr == "" {
		return nil, nil
	}

	var demos []DemoSpec
	specs := strings.Split(customDemosStr, ";")

	for i, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}

		var name, args string
		if colonIndex := strings.Index(spec, ":"); colonIndex > 0 {
			name = strings.TrimSpace(spec[:colonIndex])
			args = strings.TrimSpace(spec[colonIndex+1:])
		} else {
			name = fmt.Sprintf("custom-%d", i+1)
			args = spec
		}

		if args == "" {
			return nil, fmt.Errorf("empty args for custom demo: %s", spec)
		}

		directory := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), " ", "-"), ":", "-")
		demos = append(demos, DemoSpec{
			Name:        name,
			Title:       fmt.Sprintf("Custom: %s", name),
			Description: fmt.Sprintf("Custom demo: %s", args),
			Args:        args,
			Directory:   directory,
		})
	}

	return demos, nil
}

func createS3BucketIfNotExists(config *DemoConfig) error {
	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return err
	}

	if err := exec.Command("aws", "s3", "ls", fmt.Sprintf("s3://%s", config.S3Bucket)).Run(); err == nil {
		logf("S3 bucket already exists: %s", config.S3Bucket)
		return nil
	}
	logf("Creating S3 bucket: %s", config.S3Bucket)
	var createCmd *exec.Cmd
	if config.S3Region == "us-east-1" {
		createCmd = exec.Command("aws", "s3", "mb", fmt.Sprintf("s3://%s", config.S3Bucket))
	} else {
		createCmd = exec.Command("aws", "s3", "mb", fmt.Sprintf("s3://%s", config.S3Bucket), "--region", config.S3Region)
	}

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("creating S3 bucket: %w", err)
	}

	websiteCmd := exec.Command("aws", "s3", "website", fmt.Sprintf("s3://%s", config.S3Bucket),
		"--index-document", "index.html", "--error-document", "error.html")
	websiteCmd.Run()

	exec.Command("aws", "s3api", "delete-public-access-block", "--bucket", config.S3Bucket).Run()
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Sid": "PublicReadGetObject",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::%s/*"
		}]
	}`, config.S3Bucket)

	if tempFile, err := os.CreateTemp("", "policy-*.json"); err == nil {
		tempFile.WriteString(policy)
		tempFile.Close()
		exec.Command("aws", "s3api", "put-bucket-policy", "--bucket", config.S3Bucket,
			"--policy", fmt.Sprintf("file://%s", tempFile.Name())).Run()
		os.Remove(tempFile.Name())
	}

	return nil
}

func uploadSampleDataToS3(config *DemoConfig) error {
	logf("Uploading sample data to S3...")

	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return err
	}

	dataS3Path := fmt.Sprintf("s3://%s/data/", config.S3Bucket)
	syncCmd := exec.Command("aws", "s3", "sync", config.DemoDataDir, dataS3Path)

	if err := syncCmd.Run(); err != nil {
		return fmt.Errorf("uploading data to S3: %w", err)
	}

	return nil
}

func uploadErrorPageToS3(config *DemoConfig) error {
	logf("Uploading error page to S3...")

	if err := validateS3BucketName(config.S3Bucket); err != nil {
		return err
	}

	errorPagePath := filepath.Join(config.TemplatesDir, "error.html")
	if _, err := os.Stat(errorPagePath); os.IsNotExist(err) {
		return fmt.Errorf("error page template not found: %s", errorPagePath)
	}

	uploadCmd := exec.Command("aws", "s3", "cp", errorPagePath, fmt.Sprintf("s3://%s/error.html", config.S3Bucket))
	if err := uploadCmd.Run(); err != nil {
		return fmt.Errorf("uploading error page to S3: %w", err)
	}

	logf("Error page uploaded successfully")
	return nil
}

func trackS3Bucket(config *DemoConfig) error {
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")

	var buckets []BucketRecord
	if data, err := os.ReadFile(bucketFile); err == nil {
		json.Unmarshal(data, &buckets)
	}

	record := BucketRecord{
		Name:      config.S3Bucket,
		Region:    config.S3Region,
		Created:   time.Now(),
		ConfigDir: filepath.Dir(config.ConfigFile),
	}

	found := false
	for i, bucket := range buckets {
		if bucket.Name == config.S3Bucket {
			buckets[i] = record
			found = true
			break
		}
	}
	if !found {
		buckets = append(buckets, record)
	}

	data, _ := json.MarshalIndent(buckets, "", "  ")
	return os.WriteFile(bucketFile, data, 0o600)
}

func getTrackedBuckets(config *DemoConfig) ([]BucketRecord, error) {
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")

	var buckets []BucketRecord
	data, err := os.ReadFile(bucketFile)
	if err != nil {
		return buckets, nil
	}

	json.Unmarshal(data, &buckets)
	return buckets, nil
}

func removeTrackedBucket(config *DemoConfig, bucketName string) error {
	buckets, err := getTrackedBuckets(config)
	if err != nil {
		return err
	}

	var filtered []BucketRecord
	for _, bucket := range buckets {
		if bucket.Name != bucketName {
			filtered = append(filtered, bucket)
		}
	}

	data, _ := json.MarshalIndent(filtered, "", "  ")
	bucketFile := filepath.Join(config.ProjectRoot, ".demo-buckets.json")
	return os.WriteFile(bucketFile, data, 0o600)
}

func validateS3BucketName(bucket string) error {
	if bucket == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`, bucket)
	if !matched || len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("invalid S3 bucket name: %s", bucket)
	}

	return nil
}

func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type TreeItem struct {
	Name        string
	IsDir       bool
	Size        string
	Depth       int
	IsNoIndex   bool
	IsSkipIndex bool
	Indent      string
	Prefix      string
	Class       string
	Suffix      string
}

type SourceTreeData struct {
	Items []TreeItem
}

func generateSourceTree(config *DemoConfig) error {
	logf("Generating source data tree...")

	var items []TreeItem

	err := filepath.Walk(config.DemoDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path and depth
		relPath, err := filepath.Rel(config.DemoDataDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		size := ""
		if !info.IsDir() {
			size = humanizeBytes(info.Size())
		}

		// Prepare template data
		indent := strings.Repeat("    ", depth)
		prefix := "├── "
		class := "file"
		suffix := ""

		if info.IsDir() {
			class = "dir"
			suffix = "/"
		} else if info.Name() == ".noindex" {
			class = "special"
			suffix = fmt.Sprintf(" (%s) [excludes directory from indexing]", size)
		} else if info.Name() == ".skipindex" {
			class = "special"
			suffix = fmt.Sprintf(" (%s) [directory appears in listings but no index generated]", size)
		} else if size != "" {
			suffix = fmt.Sprintf(" (%s)", size)
		}

		item := TreeItem{
			Name:        info.Name(),
			IsDir:       info.IsDir(),
			Size:        size,
			Depth:       depth,
			IsNoIndex:   info.Name() == ".noindex",
			IsSkipIndex: info.Name() == ".skipindex",
			Indent:      indent,
			Prefix:      prefix,
			Class:       class,
			Suffix:      suffix,
		}

		items = append(items, item)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walking source directory: %w", err)
	}

	// Generate HTML using template
	templatePath := filepath.Join(config.TemplatesDir, "source-tree.html.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parsing source tree template: %w", err)
	}

	sourceTreePath := filepath.Join(config.DemoDataDir, "source-tree.html")
	file, err := os.Create(sourceTreePath)
	if err != nil {
		return fmt.Errorf("creating source tree file: %w", err)
	}
	defer file.Close()

	data := SourceTreeData{Items: items}
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("executing source tree template: %w", err)
	}

	logf("Generated source tree: %s", sourceTreePath)
	return nil
}
