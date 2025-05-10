// Simple program to generate index files for a directory or S3 bucket.
package webindexer

import (
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/charmbracelet/log"
)

//go:embed templates/themes/default.html.tmpl
var defaultTemplate string

//go:embed templates/themes/solarized.html.tmpl
var solarizedTemplate string

//go:embed templates/themes/nord.html.tmpl
var nordTemplate string

//go:embed templates/themes/dracula.html.tmpl
var draculaTemplate string

// Indexer is the main struct for the webindexer package.
type Indexer struct {
	Cfg          Config
	Source       FileSource
	Target       FileSource
	s3           *s3.S3
	BackendSetup BackendSetup
}

// FileSource is an interface for listing the contents of a directory or S3
// bucket.
type FileSource interface {
	Read(path string) ([]Item, bool, error)
	Write(data Data, content string) error
	EnsureDirExists(relativePath string) error
}

// Item represents an S3 key, or a local file/directory.
type Item struct {
	Name         string
	Size         string
	LastModified string
	URL          string
	IsDir        bool
	Items        []Item
}

// Data holds the template data.
type Data struct {
	Title        string
	Path         string
	RootPath     string
	RelativePath string
	URL          string
	Items        []Item
	Parent       string
	HasParent    bool
	ParentURL    string
}

type BackendSetup interface {
	Setup(indexer *Indexer) error
}

type defaultBackendSetup struct{}

func (d defaultBackendSetup) Setup(indexer *Indexer) error {
	return setupBackends(indexer)
}

// New creates a new Indexer, taking the initial configuration and returning a
// updating it with the service, source and target paths.
func New(cfg Config) (*Indexer, error) {
	indexer := &Indexer{
		Cfg:          cfg,
		BackendSetup: defaultBackendSetup{},
	}

	if err := indexer.Cfg.Validate(); err != nil {
		return nil, err
	}

	if err := indexer.BackendSetup.Setup(indexer); err != nil {
		return nil, err
	}

	return indexer, nil
}

func joinURL(baseURL string, parts ...string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	joinedPath := path.Join(parts...)
	u.Path = path.Join(u.Path, joinedPath)

	return u.String(), nil
}

func resolveParentPath(baseURL, parent, indexFile string, linkToIndexes bool) string {
	parentURL, err := joinURL(baseURL, parent)
	if err != nil {
		log.Error("Error joining URL:", err)
	}

	if linkToIndexes {
		parentURL += indexFile
	}
	return parentURL
}

func resolveItemURL(baseURL, path, name string, isDir, linkToIndexes bool, indexFile string) string {
	var err error
	url := name
	if baseURL != "" {
		url, err = joinURL(baseURL, path, name)
		if err != nil {
			log.Error("Error joining URL:", err)
		}
	}

	if isDir {
		url = strings.TrimSuffix(url, "/") + "/"
		if linkToIndexes {
			url += indexFile
		}
	}
	return url
}

// setupBackends sets up the source and target backends for the indexer.
func setupBackends(indexer *Indexer) error {
	var err error

	if isS3URI(indexer.Cfg.Source) || isS3URI(indexer.Cfg.Target) {
		log.Debug("Setting up S3 session")
		cfg := aws.NewConfig()
		if indexer.Cfg.S3Endpoint != "" {
			cfg = cfg.WithEndpoint(indexer.Cfg.S3Endpoint)
		}
		sess, err := session.NewSession(cfg)
		if err != nil {
			return fmt.Errorf("failed to create AWS session: %w", err)
		}

		indexer.s3 = s3.New(sess)
	}

	// For local directories, convert relative paths to absolute paths
	if !isS3URI(indexer.Cfg.Source) {
		absPath, err := filepath.Abs(indexer.Cfg.Source)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for source: %w", err)
		}
		log.Debugf("Converted source path from %s to %s", indexer.Cfg.Source, absPath)
		indexer.Cfg.Source = absPath
	}

	indexer.Cfg.BasePath = strings.TrimSuffix(indexer.Cfg.Source, "/")
	if isS3URI(indexer.Cfg.Source) {
		_, prefix := uriToBucketAndPrefix(indexer.Cfg.Source)
		if prefix == "" {
			indexer.Cfg.BasePath = "/"
		} else {
			indexer.Cfg.BasePath = prefix
		}
	}

	indexer.Source, err = setupBackend(indexer.Cfg.Source, indexer)
	if err != nil {
		return err
	}

	indexer.Target, err = setupBackend(indexer.Cfg.Target, indexer)
	if err != nil {
		return err
	}

	return nil
}

// setupBackend sets up the backend for the given URI.
func setupBackend(uri string, indexer *Indexer) (FileSource, error) {
	log.Debugf("Setting up backend for %s", uri)
	if isS3URI(uri) {
		bucket, _ := uriToBucketAndPrefix(uri)
		return &S3Backend{svc: indexer.s3, bucket: bucket, cfg: indexer.Cfg}, nil
	}
	return &LocalBackend{path: uri, cfg: indexer.Cfg}, nil
}

// Generate the index file for the given path.
func (i Indexer) Generate(path string) error {
	var err error

	items, hasNoIndex, err := i.Source.Read(path)
	if err != nil {
		return err
	}

	// If hasNoIndex is true, skip this directory entirely
	if hasNoIndex {
		log.Debugf("Skipping generation for %s due to noindex file", path)
		return nil
	}

	// Prepare template data regardless of whether items were found
	data, err := i.data(items, path)
	if err != nil {
		return err
	}

	// Ensure the target directory exists before attempting to write or recurse
	if err := i.Target.EnsureDirExists(data.RelativePath); err != nil {
		return fmt.Errorf("failed to ensure target directory exists for %s: %w", data.RelativePath, err)
	}

	// Only generate and write the index file if there are items to list.
	// This handles the skipindex case (Read returns empty items) and empty directories.
	if len(items) > 0 {
		var tmpl *template.Template
		var templStr string
		if i.Cfg.Template != "" {
			log.Debugf("Using custom template %s for %s", i.Cfg.Template, path)
			templBytes, err := os.ReadFile(i.Cfg.Template)
			if err != nil {
				return err
			}
			templStr = string(templBytes)
		} else {
			log.Debugf("Using %s theme template for %s", i.Cfg.Theme, path)
			templStr = getThemeTemplate(i.Cfg.Theme)
		}

		tmpl, err = template.New("index").Parse(templStr)
		if err != nil {
			return err
		}

		generated := new(strings.Builder)
		if err := tmpl.Execute(generated, data); err != nil {
			return err
		}

		output := generated.String()
		if i.Cfg.Minify {
			output = minifyHTML(generated.String())
		}

		if err := i.Target.Write(data, output); err != nil {
			return err
		}
	} else {
		// Log if we are skipping the write due to empty items (skipindex or empty dir)
		log.Debugf("Skipping index file generation for %s (no items or skipindex found)", path)
	}

	// Process items to handle recursion.
	// This loop won't execute if items is empty.
	for _, item := range items { // Iterate over original items
		err := i.parseItem(path, item) // Pass item by value, check error
		if err != nil {
			// Stop processing if any subdirectory fails? Or just log?
			// Return the error to propagate it up.
			return err
		}
	}

	return nil
}

// getThemeTemplate returns the template string for the given theme.
func getThemeTemplate(theme string) string {
	switch theme {
	case "solarized":
		return solarizedTemplate
	case "nord":
		return nordTemplate
	case "dracula":
		return draculaTemplate
	default:
		return defaultTemplate
	}
}

func (i Indexer) data(items []Item, path string) (Data, error) {
	relativePath := strings.TrimPrefix(path, i.Cfg.BasePath)

	// Ensure relative path is prefixed with a slash. This will also set an
	// empty base path to "/" (such as when listing the root of an S3 bucket).
	// S3 keys don't have a leading slash, but we normalize for consistency
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}

	data := Data{
		RootPath:     i.Cfg.BasePath,
		Items:        make([]Item, 0, len(items)),
		Path:         path,
		RelativePath: relativePath,
		URL:          i.Cfg.BaseURL,
		Title:        i.formatTitle(path, relativePath),
	}

	if path == i.Cfg.BasePath {
		data.HasParent = false
	} else {
		parent := filepath.Dir(path)
		// Calculate the relative parent path
		relativeParent := strings.TrimPrefix(parent, i.Cfg.BasePath)
		if !strings.HasPrefix(relativeParent, "/") {
			relativeParent = "/" + relativeParent
		}
		data.Parent = resolveParentPath(i.Cfg.BaseURL, relativeParent, i.Cfg.IndexFile, i.Cfg.LinkToIndexes)
		data.HasParent = parent != path
		if data.HasParent {
			data.ParentURL = resolveItemURL(i.Cfg.BaseURL, relativeParent, "..", true, i.Cfg.LinkToIndexes, i.Cfg.IndexFile)
		}
	}

	// Process items within the data function to set their URLs
	processedItems := make([]Item, 0, len(items))
	for _, item := range items {
		processedItem, err := i.processItemForData(path, item) // Rename to avoid confusion with recursive call
		if err != nil {
			return Data{}, err
		}
		processedItems = append(processedItems, processedItem)
	}
	data.Items = processedItems // Assign processed items with URLs

	i.sort(&data.Items)
	return data, nil
}

// processItemForData generates the URL for an item. Does NOT handle recursion.
func (i Indexer) processItemForData(path string, item Item) (Item, error) {
	// Calculate the relative path by removing the base path
	relativePath := strings.TrimPrefix(path, i.Cfg.BasePath)
	// Ensure relative path is prefixed with a slash
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}

	item.URL = resolveItemURL(i.Cfg.BaseURL, relativePath, item.Name, item.IsDir, i.Cfg.LinkToIndexes, i.Cfg.IndexFile)
	// Return the item with the URL set
	return item, nil
}

// parseItem handles the recursive call for directories.
func (i Indexer) parseItem(path string, item Item) error {
	// If the item is a directory and recursive mode is enabled, generate its index
	if item.IsDir && i.Cfg.Recursive {
		// Construct the full path for the subdirectory
		subDirPath := filepath.Join(path, item.Name)
		if err := i.Generate(subDirPath); err != nil {
			// Log the error but also return it to stop processing this branch
			log.Errorf("Error generating index for subdirectory %s: %v", subDirPath, err)
			return fmt.Errorf("error generating index for subdirectory %s: %w", subDirPath, err)
		}
	}
	return nil // Return nil error if no recursion error occurred
}

func (i Indexer) formatTitle(path, relativePath string) string {
	title := strings.Replace(i.Cfg.Title, "{source}", filepath.Base(i.Cfg.Source), -1)
	title = strings.Replace(title, "{target}", filepath.Base(i.Cfg.Target), -1)
	title = strings.Replace(title, "{path}", path, -1)
	title = strings.Replace(title, "{relativePath}", relativePath, -1)

	return title
}

func humanizeBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	if bytes < 10 {
		return fmt.Sprintf("%d B", bytes)
	}
	log := math.Log(float64(bytes)) / math.Log(1024)
	index := int(log)
	size := float64(bytes) / math.Pow(1024, float64(index))
	return fmt.Sprintf("%.2f %s", size, units[index])
}

func minifyHTML(str string) string {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\t", "")
	str = strings.ReplaceAll(str, "  ", "")
	str = strings.ReplaceAll(str, "> <", "><")
	return str
}

func shouldSkip(name, index string, skips []string) bool {
	if strings.HasSuffix(name, index) {
		return true
	}

	if contains(skips, name) {
		return true
	}

	return false
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
