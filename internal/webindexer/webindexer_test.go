package webindexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBackendSetup struct {
	mock.Mock
}

type MockSource struct {
	mock.Mock
}

func (m *MockSource) Read(path string) ([]*Item, bool, error) {
	args := m.Called(path)
	return args.Get(0).([]*Item), args.Bool(1), args.Error(2)
}

func (m *MockSource) Write(data Data, content string) error {
	args := m.Called(data, content)
	return args.Error(0)
}

func (m *MockSource) EnsureDirExists(relativePath string) error {
	args := m.Called(relativePath)
	return args.Error(0)
}

func TestIndexer_Generate(t *testing.T) {
	mockSource := new(MockSource)
	mockTarget := new(MockSource)
	indexer := Indexer{
		Source: mockSource,
		Target: mockTarget,
		Cfg: Config{
			Minify: true,
		},
	}

	mockSource.On("Read", mock.Anything).Return([]*Item{}, false, nil)
	// Expect EnsureDirExists to be called on the target
	mockTarget.On("EnsureDirExists", mock.AnythingOfType("string")).Return(nil)
	// Write should NOT be called when Read returns empty items
	// mockTarget.On("Write", mock.Anything, mock.Anything).Return(nil)

	err := indexer.Generate(nil, "path/to/generate")
	assert.NoError(t, err)

	mockSource.AssertExpectations(t)
	mockTarget.AssertExpectations(t)
}

func TestCustomTemplate(t *testing.T) {
	mockSource := new(MockSource)
	mockTarget := new(MockSource)

	// Create a temporary file for the custom template
	file, err := os.CreateTemp("", "template.html")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	// Write the custom template to the file
	html := "<html><head><title>Custom Template</title></head></html>"
	_, err = file.WriteString(html)
	require.NoError(t, err)

	// Close the file
	err = file.Close()
	require.NoError(t, err)

	// Create an indexer with the custom template
	indexer := Indexer{
		Source: mockSource,
		Target: mockTarget,
		Cfg: Config{
			Template: file.Name(),
			Minify:   true,
		},
	}

	require.NoError(t, err)

	mockSource.On("Read", mock.Anything).Return([]*Item{}, false, nil)
	// Expect EnsureDirExists to be called on the target before writing
	mockTarget.On("EnsureDirExists", mock.AnythingOfType("string")).Return(nil)
	// Write should NOT be called when Read returns empty items
	// mockTarget.On("Write", mock.Anything, mock.Anything).Return(nil)

	err = indexer.Generate(nil, "path/to/generate")
	assert.NoError(t, err)

	// Check the file content
	f, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	assert.Equal(t, html, string(f))

	mockSource.AssertExpectations(t)
	mockTarget.AssertExpectations(t)
}

func (m *MockBackendSetup) Setup(indexer *Indexer) error {
	args := m.Called(indexer)
	return args.Error(0)
}

func TestNew(t *testing.T) {
	mockSetup := new(MockBackendSetup)
	cfg := Config{
		Source: "s3://mybucket/some/source/path",
		Target: "/mnt/some/target/path",
		SortBy: "name",
		Order:  "asc",
	}

	mockSetup.On("Setup", mock.Anything).Return(nil)

	indexer, err := New(cfg)
	require.NoError(t, err)
	indexer.BackendSetup = mockSetup

	err = indexer.BackendSetup.Setup(indexer)
	require.NoError(t, err)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Source: "some/source/path",
				Target: "some/target/path",
				SortBy: "name",
				Order:  "asc",
				Minify: true,
			},
			wantErr: false,
		},
		{
			name: "invalid config - missing source",
			config: Config{
				Target: "some/target/path",
				SortBy: "name",
				Order:  "asc",
			},
			wantErr: true,
		},
		{
			name: "invalid config - missing target",
			config: Config{
				Source: "some/source/path",
				SortBy: "name",
				Order:  "asc",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMinifyHTML(t *testing.T) {
	html := `
<html>
  <head>
    <title>Test</title>
  </head>
</html>
`
	minified := minifyHTML(html)

	// The minified HTML should not contain any newlines
	assert.False(t, strings.Contains(minified, "\n"))

	// The minified HTML should not contain any leading or trailing whitespace
	assert.False(t, strings.HasPrefix(minified, " "))
	assert.False(t, strings.HasSuffix(minified, " "))

	// The minified HTML should not contain any whitespace between tags
	assert.False(t, strings.Contains(minified, "> <"))
}

func TestShouldSkipURL(t *testing.T) {
	// URLs that should be skipped
	skips := []string{
		"index.html",
		"index.htm",
		"foo.txt",
	}

	assert.True(t, shouldSkip("index.html", "index.html", skips))
	assert.True(t, shouldSkip("index.htm", "index.htm", skips))
	assert.True(t, shouldSkip("foo.txt", "index.html", skips))
	assert.False(t, shouldSkip("another-file.tar.gz", "index.html", skips))
	assert.False(t, shouldSkip("something.html", "index.html", skips))
}

func TestResolveParentPath(t *testing.T) {
	tests := []struct {
		baseURL       string
		parent        string
		indexFile     string
		linkToIndexes bool
		expected      string
	}{
		{"", "dir", "index.html", false, "dir"},
		{"/blah", "dir", "index.html", false, "/blah/dir"},
		{"https://foo.com/one/two", "dir", "index.html", false, "https://foo.com/one/two/dir"},
		{"https://foo.com/one/two/", "dir", "index.html", false, "https://foo.com/one/two/dir"},
	}

	for _, test := range tests {
		result := resolveParentPath(test.baseURL, test.parent, test.indexFile, test.linkToIndexes)
		assert.Equal(t, test.expected, result)
	}
}

func TestResolveItemURL(t *testing.T) {
	tests := []struct {
		baseURL       string
		path          string
		name          string
		isDir         bool
		linkToIndexes bool
		indexFile     string
		expected      string
	}{
		{"https://foo.com", "one/two", "three", false, false, "", "https://foo.com/one/two/three"},
		{"https://foo.com", "one/two", "three/", true, false, "", "https://foo.com/one/two/three/"},
		{"https://foo.com", "one/two", "three", true, true, "index.html", "https://foo.com/one/two/three/index.html"},
		// Test without base URL
		{"", "one/two", "three", false, false, "", "three"},
		// Test with empty path
		{"https://foo.com", "", "three", false, false, "", "https://foo.com/three"},
		// Error scenario can be mocked if joinURL is mockable or by causing it to fail with specific input
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveItemURL(tc.baseURL, tc.path, tc.name, tc.isDir, tc.linkToIndexes, tc.indexFile)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIndexer_ProcessItemForData tests the processItemForData method
func TestIndexer_ProcessItemForData(t *testing.T) {
	// Create some temporary directories and files
	sourceDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir) // Clean up after the test

	// Create mock files (content doesn't matter for this test)
	fileNames := []string{"file1.txt"}
	for _, fName := range fileNames {
		tmpFn := filepath.Join(sourceDir, fName)
		if err := os.WriteFile(tmpFn, []byte("test content"), 0o666); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}
	// Create a mock directory
	dirName := "dir"
	if err := os.Mkdir(filepath.Join(sourceDir, dirName), 0o755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// No need for targetDir in this specific test

	indexer := Indexer{
		Cfg: Config{
			BaseURL:       "https://example.com",
			Source:        sourceDir,      // Source is the temp dir
			Target:        "/fake/target", // Target doesn't matter here
			LinkToIndexes: true,
			IndexFile:     "index.html",
			Recursive:     true,      // Recursive doesn't affect processItemForData directly
			BasePath:      sourceDir, // Set BasePath correctly
		},
	}
	// No need to call setupBackends as we manually set BasePath

	// Simulate items
	itemDir := &Item{Name: "dir", IsDir: true}
	itemFile1 := &Item{Name: "file1.txt", IsDir: false}

	// Test directory item
	modifiedItemDir, err := indexer.processItemForData(sourceDir, itemDir)
	require.NoError(t, err)
	expectUrlDir := "https://example.com/dir/index.html" // Expect URL relative to BasePath
	assert.Equal(t, expectUrlDir, modifiedItemDir.URL)

	// Test file item
	modifiedItemFile, err := indexer.processItemForData(sourceDir, itemFile1)
	require.NoError(t, err)
	expectUrlFile := "https://example.com/file1.txt" // Expect URL relative to BasePath
	assert.Equal(t, expectUrlFile, modifiedItemFile.URL)
}

func TestGetThemeTemplate(t *testing.T) {
	tests := []struct {
		name     string
		theme    string
		expected string
	}{
		{
			name:     "default theme",
			theme:    "default",
			expected: defaultTemplate,
		},
		{
			name:     "solarized theme",
			theme:    "solarized",
			expected: solarizedTemplate,
		},
		{
			name:     "nord theme",
			theme:    "nord",
			expected: nordTemplate,
		},
		{
			name:     "dracula theme",
			theme:    "dracula",
			expected: draculaTemplate,
		},
		{
			name:     "unknown theme",
			theme:    "unknown",
			expected: defaultTemplate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getThemeTemplate(tc.theme)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerate_Recursive(t *testing.T) {
	// 1. Setup source directory
	sourceDir, err := os.MkdirTemp("", "TestGenerateRecursiveSource*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)
	absSourceDir, err := filepath.Abs(sourceDir) // Need absolute path for BasePath consistency
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(absSourceDir, "file1.txt"), []byte("content1"), 0o644)
	require.NoError(t, err)
	subDir := filepath.Join(absSourceDir, "subdir")
	err = os.Mkdir(subDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0o644)
	require.NoError(t, err)

	// 2. Setup target mock
	mockTarget := new(MockSource) // Reusing MockSource for target interface compliance

	// 3. Create Indexer directly (bypass New and setupBackends for cleaner mocking)
	cfg := Config{
		Source:         absSourceDir,              // Use absolute path
		Target:         "/fake/target",            // Mock target path
		Recursive:      true,                      // Enable recursion
		SortBy:         "name",                    // Define required config fields
		Order:          "asc",                     // Define required config fields
		IndexFile:      "index.html",              // Use standard index file name
		BasePath:       absSourceDir,              // Set BasePath explicitly
		DateFormat:     "2006-01-02 15:04:05 MST", // Provide default or required format
		DirsFirst:      true,                      // Standard behavior
		NoIndexFiles:   []string{},                // Ensure no skips interfere
		SkipIndexFiles: []string{},                // Ensure no skips interfere
		Skips:          []string{},                // Ensure no skips interfere
		Theme:          "default",                 // Provide default theme
	}
	sourceBackend := &LocalBackend{path: absSourceDir, cfg: cfg} // Use real local backend for source

	indexer := Indexer{
		Cfg:    cfg,
		Source: sourceBackend,
		Target: mockTarget, // Use the mock target
	}

	// 4. Setup mock expectations
	// Expect EnsureDirExists for root ("/") and subdir ("/subdir") relative paths
	// Allow any number of calls because Write might also call it internally depending on implementation.
	mockTarget.On("EnsureDirExists", mock.AnythingOfType("string")).Return(nil)

	// Expect Write for root index: should contain file1.txt and subdir/
	// We verify the RelativePath and that the 'subdir' item exists.
	mockTarget.On("Write", mock.MatchedBy(func(data Data) bool {
		if data.RelativePath != "/" {
			return false
		}
		foundSubdir := false
		for _, item := range data.Items {
			if item.Name == "subdir" && item.IsDir {
				foundSubdir = true
				break
			}
		}
		// Also check for file1.txt
		foundFile1 := false
		for _, item := range data.Items {
			if item.Name == "file1.txt" && !item.IsDir {
				foundFile1 = true
				break
			}
		}
		return foundSubdir && foundFile1 && len(data.Items) == 2
	}), mock.AnythingOfType("string")).Return(nil).Once()

	// Expect Write for subdir index: should contain file2.txt
	// We verify the RelativePath and that 'file2.txt' exists.
	mockTarget.On("Write", mock.MatchedBy(func(data Data) bool {
		if data.RelativePath != "/subdir" {
			return false
		}
		return len(data.Items) == 1 && data.Items[0].Name == "file2.txt" && !data.Items[0].IsDir
	}), mock.AnythingOfType("string")).Return(nil).Once()

	// 5. Call Generate from the root source path
	err = indexer.Generate(nil, absSourceDir)
	require.NoError(t, err)

	// 6. Assert mock expectations were met
	mockTarget.AssertExpectations(t)
}

func TestLinkUpFromRoot(t *testing.T) {
	// Test the "link up from root" functionality
	sourceDir, err := os.MkdirTemp("", "TestLinkUpFromRoot*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a test file
	err = os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0o644)
	require.NoError(t, err)

	// Test with LinkUpFromRoot enabled
	cfg := Config{
		Source:         sourceDir,
		BasePath:       sourceDir,
		LinkUpFromRoot: true,
		LinkUpText:     "Custom Up Text",
		LinkUpURL:      "../custom",
		LinkToIndexes:  false,
		BaseURL:        "https://example.com",
		IndexFile:      "index.html",
	}

	sourceBackend := &LocalBackend{path: sourceDir, cfg: cfg}
	indexer := Indexer{
		Cfg:    cfg,
		Source: sourceBackend,
	}

	// Generate data for the root path
	items, _, err := sourceBackend.Read(sourceDir)
	require.NoError(t, err)

	data, err := indexer.data(items, sourceDir, sourceDir)
	require.NoError(t, err)

	// Verify that HasParent is true and ParentURL/ParentText are set correctly
	assert.True(t, data.HasParent, "HasParent should be true when LinkUpFromRoot is enabled")
	assert.Equal(t, "Custom Up Text", data.ParentText, "ParentText should match config")
	assert.Equal(t, "https://example.com/../custom", data.ParentURL, "ParentURL should be built correctly")

	// Test with LinkToIndexes enabled
	cfg.LinkToIndexes = true
	indexer.Cfg = cfg

	data, err = indexer.data(items, sourceDir, sourceDir)
	require.NoError(t, err)

	// Verify that the index file is appended when LinkToIndexes is true
	assert.Equal(
		t,
		"https://example.com/../custom/index.html",
		data.ParentURL,
		"ParentURL should include index file when LinkToIndexes is true",
	)

	// Test with LinkUpFromRoot disabled
	cfg.LinkUpFromRoot = false
	indexer.Cfg = cfg

	data, err = indexer.data(items, sourceDir, sourceDir)
	require.NoError(t, err)

	// Verify that HasParent is false when LinkUpFromRoot is disabled
	assert.False(t, data.HasParent, "HasParent should be false when LinkUpFromRoot is disabled")
	assert.Empty(t, data.ParentURL, "ParentURL should be empty when LinkUpFromRoot is disabled")
}

func TestSetupBackends_CustomS3Endpoint(t *testing.T) {
	tests := []struct {
		name       string
		s3Endpoint string
		expected   string
	}{
		{
			name:       "url with scheme",
			s3Endpoint: "https://custom-s3-endpoint",
			expected:   "https://custom-s3-endpoint",
		},
		{
			name:       "url without scheme",
			s3Endpoint: "custom-s3-endpoint",
			expected:   "https://custom-s3-endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()

			indexer := &Indexer{
				Cfg: Config{
					Source:     "s3://test-bucket/source/",
					S3Endpoint: tc.s3Endpoint,
				},
			}

			err := setupBackends(indexer)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, *indexer.s3.Options().BaseEndpoint)
		})
	}
}
