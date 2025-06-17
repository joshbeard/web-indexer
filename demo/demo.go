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
	Type          string
	Serve         bool
	Cleanup       bool
	ProjectRoot   string
	DemoDataDir   string
	DemoOutputDir string
	TemplatesDir  string
	S3Bucket      string
	S3Region      string
	S3PublicURL   string
	Config        *Config
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
	Name    string    `json:"name"`
	Region  string    `json:"region"`
	Created time.Time `json:"created"`
}

func main() {
	demoType := flag.String("type", "local", "Demo type: local, s3, both, s3-release")
	serve := flag.Bool("serve", false, "Start web server to preview demo")
	cleanup := flag.Bool("cleanup", false, "Clean up demo files")
	configFile := flag.String("config", "config.yml", "Path to YAML configuration file")
	customDemos := flag.String("custom-demos", "", "Semicolon-separated custom demo specs")
	help := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	config := &DemoConfig{
		Type:    *demoType,
		Serve:   *serve,
		Cleanup: *cleanup,
	}

	if err := setupConfig(config, *configFile, *customDemos); err != nil {
		log.Fatalf("Setup error: %v", err)
	}

	if config.Cleanup {
		cleanupDemo(config)
		return
	}

	logf("Starting demo: %s", config.Type)
	if err := runDemo(config); err != nil {
		log.Fatalf("Demo error: %v", err)
	}

	if config.Serve {
		serveDemo(config)
	}
}

func showHelp() {
	fmt.Printf(`Web-Indexer Demo Generator

Usage: %s [options]

Options:
  -type string        Demo type: local, s3, both, s3-release (default "local")
  -serve             Start web server to preview demo
  -cleanup           Clean up demo files
  -config string     Path to YAML configuration file (default "config.yml")
  -custom-demos string   Semicolon-separated custom demo specs
  -help              Show this help

Examples:
  %s -serve
  %s -type s3
  %s -type both -serve
  %s -cleanup
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func setupConfig(config *DemoConfig, configFile, customDemos string) error {
	wd, _ := os.Getwd()

	if filepath.Base(wd) == "demo" {
		config.ProjectRoot = filepath.Dir(wd)
		config.DemoDataDir = filepath.Join(wd, "data")
		config.DemoOutputDir = filepath.Join(wd, "output")
		config.TemplatesDir = filepath.Join(wd, "templates")
		if !filepath.IsAbs(configFile) {
			configFile = filepath.Join(wd, configFile)
		}
	} else {
		config.ProjectRoot = wd
		demoDir := filepath.Join(wd, "demo")
		config.DemoDataDir = filepath.Join(demoDir, "data")
		config.DemoOutputDir = filepath.Join(demoDir, "output")
		config.TemplatesDir = filepath.Join(demoDir, "templates")
		if !filepath.IsAbs(configFile) {
			configFile = filepath.Join(demoDir, configFile)
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	yamlConfig := &Config{}
	if err := yaml.Unmarshal(data, yamlConfig); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	for i, demo := range yamlConfig.Demos {
		if demo.Name == "" || demo.Args == "" || demo.Directory == "" {
			return fmt.Errorf("demo %d missing required fields", i)
		}
		if demo.Title == "" {
			yamlConfig.Demos[i].Title = demo.Name
		}
	}

	if customDemos != "" {
		customSpecs := parseCustomDemos(customDemos)
		yamlConfig.Demos = append(customSpecs, yamlConfig.Demos...)
		logf("Added %d custom demo(s)", len(customSpecs))
	}

	config.Config = yamlConfig

	if strings.Contains(config.Type, "s3") {
		return setupS3Config(config)
	}
	return nil
}

func setupS3Config(config *DemoConfig) error {
	if bucket := os.Getenv("DEMO_S3_BUCKET"); bucket != "" {
		config.S3Bucket = bucket
	} else if config.Type == "s3-release" {
		return fmt.Errorf("DEMO_S3_BUCKET required for release preview")
	} else {
		config.S3Bucket = fmt.Sprintf("%s-%d", config.Config.S3.BucketPrefix, time.Now().Unix())
	}

	if !regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`).MatchString(config.S3Bucket) ||
		len(config.S3Bucket) < 3 || len(config.S3Bucket) > 63 {
		return fmt.Errorf("invalid S3 bucket name: %s", config.S3Bucket)
	}

	config.S3Region = getEnvOrDefault("AWS_REGION", config.Config.S3.Region)
	os.Setenv("AWS_REGION", config.S3Region)

	if config.Config.S3.CustomDomain != "" && os.Getenv("GITHUB_ACTIONS") == "true" {
		baseURL := strings.TrimSuffix(config.Config.S3.CustomDomain, "/")
		if config.Type == "s3-release" {
			config.S3PublicURL = fmt.Sprintf("%s/", baseURL)
		} else if prNumber := os.Getenv("PR_NUMBER"); prNumber != "" {
			config.S3PublicURL = fmt.Sprintf("%s/%s/", baseURL, prNumber)
		} else {
			config.S3PublicURL = fmt.Sprintf("%s/", baseURL)
		}
	} else {
		if config.S3Region == "us-east-1" {
			config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-us-east-1.amazonaws.com/", config.S3Bucket)
		} else {
			config.S3PublicURL = fmt.Sprintf("http://%s.s3-website-%s.amazonaws.com/", config.S3Bucket, config.S3Region)
		}
	}

	if err := exec.Command("aws", "sts", "get-caller-identity").Run(); err != nil {
		return fmt.Errorf("AWS credentials not available")
	}

	logf("S3 Config - Bucket: %s, Region: %s", config.S3Bucket, config.S3Region)
	return nil
}

func runDemo(config *DemoConfig) error {
	logf("Building web-indexer...")
	cmd := exec.Command("go", "build", "-o", "web-indexer", ".")
	cmd.Dir = config.ProjectRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if err := createDemoData(config); err != nil {
		return err
	}

	generateSourceTree(config)

	switch config.Type {
	case "local":
		return generateLocalDemo(config)
	case "s3", "s3-release":
		return generateS3Demo(config)
	case "both":
		if err := generateLocalDemo(config); err != nil {
			return err
		}
		return generateS3Demo(config)
	default:
		return fmt.Errorf("unknown demo type: %s", config.Type)
	}
}

func createDemoData(config *DemoConfig) error {
	logf("Creating demo data...")

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
	return copyFiles(templateContentDir, config.DemoDataDir)
}

func generateLocalDemo(config *DemoConfig) error {
	logf("Generating local demos...")

	for _, demo := range config.Config.Demos {
		sourceDir := config.DemoDataDir
		targetDir := filepath.Join(config.DemoOutputDir, "local", demo.Directory)

		args := []string{"--source", sourceDir, "--target", targetDir}
		if demo.Args != "" {
			customArgs := parseArgs(demo.Args)
			args = append(args, customArgs...)
		}

		if err := runWebIndexer(config, args); err != nil {
			return fmt.Errorf("demo %s failed: %w", demo.Name, err)
		}

		copyFiles(sourceDir, targetDir)
	}

	return generateIndexPage(config, "local")
}

func generateS3Demo(config *DemoConfig) error {
	logf("Generating S3 demos...")

	if err := createS3Bucket(config); err != nil {
		return err
	}

	trackS3Bucket(config)

	dataS3URI := fmt.Sprintf("s3://%s/data", config.S3Bucket)

	exec.Command("aws", "s3", "sync", config.DemoDataDir, dataS3URI+"/").Run()
	exec.Command("aws", "s3", "cp", filepath.Join(config.TemplatesDir, "error.html"),
		fmt.Sprintf("s3://%s/error.html", config.S3Bucket)).Run()

	for _, demo := range config.Config.Demos {
		targetS3URI := fmt.Sprintf("s3://%s/%s", config.S3Bucket, demo.Directory)

		args := []string{"--source", dataS3URI, "--target", targetS3URI}
		if demo.Args != "" {
			customArgs := parseArgs(demo.Args)
			args = append(args, customArgs...)
		}

		if err := runWebIndexer(config, args); err != nil {
			return fmt.Errorf("S3 demo %s failed: %w", demo.Name, err)
		}

		exec.Command("aws", "s3", "sync", dataS3URI+"/", targetS3URI+"/").Run()
	}

	exec.Command("aws", "s3", "rm", dataS3URI, "--recursive").Run()

	if err := generateS3IndexPage(config); err != nil {
		return err
	}

	logf("S3 demo complete! URL: %s", config.S3PublicURL)
	return nil
}

func generateIndexPage(config *DemoConfig, variant string) error {
	logf("Generating %s index page...", variant)

	prNumber := os.Getenv("PR_NUMBER")
	repository := os.Getenv("GITHUB_REPOSITORY")
	releaseVersion := os.Getenv("RELEASE_VERSION")

	title := "Web-Indexer Preview"
	if config.Type == "s3-release" && releaseVersion != "" {
		title = fmt.Sprintf("Web-Indexer Release Preview - v%s", releaseVersion)
	} else if prNumber != "" {
		title = fmt.Sprintf("Web-Indexer Preview - PR #%s", prNumber)
	}

	indexData := DemoIndex{
		Title:       title,
		Description: "Web-indexer generates themeable directory listings",
		Demos:       config.Config.Demos,
		PRNumber:    prNumber,
		Repository:  repository,
		CustomArgs:  os.Getenv("CUSTOM_ARGS"),
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

func generateS3IndexPage(config *DemoConfig) error {
	tempDir := filepath.Join(config.DemoOutputDir, "temp-s3-index")
	os.MkdirAll(tempDir, 0o750)
	defer os.RemoveAll(tempDir)

	originalOutputDir := config.DemoOutputDir
	config.DemoOutputDir = tempDir

	err := generateIndexPage(config, ".")
	config.DemoOutputDir = originalOutputDir

	if err != nil {
		return err
	}

	indexPath := filepath.Join(tempDir, "index.html")
	uploadCmd := exec.Command("aws", "s3", "cp", indexPath, fmt.Sprintf("s3://%s/index.html", config.S3Bucket))
	return uploadCmd.Run()
}

func cleanupDemo(config *DemoConfig) {
	logf("Cleaning up...")

	os.RemoveAll(config.DemoDataDir)
	os.RemoveAll(config.DemoOutputDir)

	if specificBucket := os.Getenv("DEMO_S3_BUCKET"); specificBucket != "" {
		deleteBucket(specificBucket)
		removeTrackedBucket(config, specificBucket)
	} else {
		buckets := getTrackedBuckets(config)
		for _, bucket := range buckets {
			deleteBucket(bucket.Name)
			removeTrackedBucket(config, bucket.Name)
		}
	}

	logf("Cleanup completed")
}

func serveDemo(config *DemoConfig) {
	port := config.Config.Server.Port
	dir := filepath.Join(config.DemoOutputDir, "local")

	logf("👉 Local Demo URL: http://localhost:%d", port)
	logf("   Serving from: %s", dir)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.FileServer(http.Dir(dir)),
	}
	server.ListenAndServe()
}

func runWebIndexer(config *DemoConfig, args []string) error {
	binaryPath := filepath.Join(config.ProjectRoot, "web-indexer")
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = config.ProjectRoot

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("web-indexer failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func parseArgs(s string) []string {
	if s == "" {
		return nil
	}

	if !strings.Contains(s, `"`) && !strings.Contains(s, `'`) {
		return strings.Fields(s)
	}

	var args []string
	var current strings.Builder
	inQuote := false
	var quoteChar byte
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		switch c {
		case '\\':
			if inQuote {
				if i+1 < len(s) && (s[i+1] == quoteChar || s[i+1] == '\\') {
					escaped = true
				} else {
					current.WriteByte(c)
				}
			} else {
				escaped = true
			}
		case '"', '\'':
			if !inQuote {
				inQuote = true
				quoteChar = c
			} else if c == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteByte(c)
			}
		case ' ', '\t', '\n':
			if inQuote {
				current.WriteByte(c)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				for i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\n') {
					i++
				}
			}
		default:
			current.WriteByte(c)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func parseCustomDemos(customDemosStr string) []DemoSpec {
	if customDemosStr == "" {
		return nil
	}

	var demos []DemoSpec
	specs := strings.Split(customDemosStr, ";")

	for i, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}

		var name, args string
		colonIndex := strings.Index(spec, ":")
		dashIndex := strings.Index(spec, "-")
		if colonIndex > 0 && (dashIndex == -1 || colonIndex < dashIndex) {
			name = strings.TrimSpace(spec[:colonIndex])
			args = strings.TrimSpace(spec[colonIndex+1:])
		} else {
			name = fmt.Sprintf("custom-%d", i+1)
			args = spec
		}

		directory := sanitizeDirectoryName(name)

		demos = append(demos, DemoSpec{
			Name:        name,
			Title:       fmt.Sprintf("Custom: %s", name),
			Description: fmt.Sprintf("Custom demo: %s", args),
			Args:        args,
			Directory:   directory,
		})
	}

	return demos
}

func sanitizeDirectoryName(name string) string {
	safe := strings.ToLower(name)
	replacer := strings.NewReplacer(
		" ", "-", ":", "-", "\"", "", "'", "", "{", "", "}", "",
		"[", "", "]", "", "(", "", ")", "", "/", "-", "\\", "-",
		"|", "-", "*", "", "?", "", "<", "", ">", "", "=", "",
		"+", "", "&", "", "%", "", "$", "", "#", "", "@", "",
		"!", "", "~", "", "`", "",
	)
	safe = replacer.Replace(safe)

	for strings.Contains(safe, "--") {
		safe = strings.ReplaceAll(safe, "--", "-")
	}

	safe = strings.Trim(safe, "-")
	if safe == "" {
		safe = "custom-demo"
	}
	if len(safe) > 50 {
		safe = safe[:47] + "..."
	}

	return safe
}

func createS3Bucket(config *DemoConfig) error {
	if exec.Command("aws", "s3", "ls", fmt.Sprintf("s3://%s", config.S3Bucket)).Run() == nil {
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

	exec.Command("aws", "s3", "website", fmt.Sprintf("s3://%s", config.S3Bucket),
		"--index-document", "index.html", "--error-document", "error.html").Run()

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

func deleteBucket(bucketName string) {
	if regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`).MatchString(bucketName) {
		exec.Command("aws", "s3", "rm", fmt.Sprintf("s3://%s", bucketName), "--recursive").Run()
		if exec.Command("aws", "s3", "rb", fmt.Sprintf("s3://%s", bucketName)).Run() == nil {
			logf("Deleted bucket: %s", bucketName)
		}
	}
}

func trackS3Bucket(config *DemoConfig) {
	bucketFile := filepath.Join(config.ProjectRoot, "demo", ".demo-buckets.json")

	var buckets []BucketRecord
	if data, err := os.ReadFile(bucketFile); err == nil {
		json.Unmarshal(data, &buckets)
	}

	record := BucketRecord{
		Name:    config.S3Bucket,
		Region:  config.S3Region,
		Created: time.Now(),
	}

	for i, bucket := range buckets {
		if bucket.Name == config.S3Bucket {
			buckets[i] = record
			goto save
		}
	}
	buckets = append(buckets, record)

save:
	if data, err := json.MarshalIndent(buckets, "", "  "); err == nil {
		os.WriteFile(bucketFile, data, 0o600)
	}
}

func getTrackedBuckets(config *DemoConfig) []BucketRecord {
	bucketFile := filepath.Join(config.ProjectRoot, "demo", ".demo-buckets.json")

	var buckets []BucketRecord
	if data, err := os.ReadFile(bucketFile); err == nil {
		json.Unmarshal(data, &buckets)
	}
	return buckets
}

func removeTrackedBucket(config *DemoConfig, bucketName string) {
	buckets := getTrackedBuckets(config)
	var filtered []BucketRecord

	for _, bucket := range buckets {
		if bucket.Name != bucketName {
			filtered = append(filtered, bucket)
		}
	}

	if data, err := json.MarshalIndent(filtered, "", "  "); err == nil {
		bucketFile := filepath.Join(config.ProjectRoot, "demo", ".demo-buckets.json")
		os.WriteFile(bucketFile, data, 0o600)
	}
}

func copyFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
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

func generateSourceTree(config *DemoConfig) {
	type TreeItem struct {
		Name   string
		IsDir  bool
		Size   string
		Depth  int
		Indent string
		Prefix string
		Class  string
		Suffix string
	}

	type SourceTreeData struct {
		Items []TreeItem
	}

	var items []TreeItem

	filepath.Walk(config.DemoDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(config.DemoDataDir, path)
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		size := ""
		if !info.IsDir() {
			size = humanizeBytes(info.Size())
		}

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

		items = append(items, TreeItem{
			Name:   info.Name(),
			IsDir:  info.IsDir(),
			Size:   size,
			Depth:  depth,
			Indent: indent,
			Prefix: prefix,
			Class:  class,
			Suffix: suffix,
		})
		return nil
	})

	templatePath := filepath.Join(config.TemplatesDir, "source-tree.html.tmpl")
	if tmpl, err := template.ParseFiles(templatePath); err == nil {
		sourceTreePath := filepath.Join(config.DemoDataDir, "source-tree.html")
		if file, err := os.Create(sourceTreePath); err == nil {
			tmpl.Execute(file, SourceTreeData{Items: items})
			file.Close()
		}
	}
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func logf(format string, args ...interface{}) {
	fmt.Printf("[DEMO] "+format+"\n", args...)
}
