package webindexer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	mockContext = mock.MatchedBy(func(ctx context.Context) bool { return true })
	mockOptFns  = mock.MatchedBy(func(optFns []func(*s3.Options)) bool { return true })
)

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) ListObjectsV2(
	context context.Context,
	params *s3.ListObjectsV2Input,
	optFns ...func(*s3.Options),
) (*s3.ListObjectsV2Output, error) {
	args := m.Called(context, params, optFns)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) PutObject(
	context context.Context,
	params *s3.PutObjectInput,
	optFns ...func(*s3.Options),
) (*s3.PutObjectOutput, error) {
	args := m.Called(context, params, optFns)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func TestS3BackendRead(t *testing.T) {
	// Arrange the test
	mockSvc := new(MockS3Client)
	backend := S3Backend{
		svc:    mockSvc,
		bucket: "test-bucket",
		cfg: Config{
			Recursive:    true,
			Source:       "s3://test-bucket",
			NoIndexFiles: []string{".noindex"},
		},
	}

	mockSvc.On(
		"ListObjectsV2",
		mockContext,
		mock.Anything,
		mockOptFns,
	).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key:          aws.String("prefix/file1.txt"),
				Size:         aws.Int64(1024),
				LastModified: aws.Time(time.Now()),
			},
			{
				Key:          aws.String("prefix/file2.txt"),
				Size:         aws.Int64(2048),
				LastModified: aws.Time(time.Now()),
			},
			{
				Key:          aws.String("prefix/smallfile1.txt"),
				Size:         aws.Int64(4),
				LastModified: aws.Time(time.Now()),
			},
			{
				Key:          aws.String("prefix/dir1/dir1file1.txt"),
				Size:         aws.Int64(2048),
				LastModified: aws.Time(time.Now()),
			},
		},
		CommonPrefixes: []types.CommonPrefix{
			{
				Prefix: aws.String("prefix/"),
			},
			{
				Prefix: aws.String("prefix/dir1/"),
			},
		},
	}, nil)

	// Test reading root directory
	items, hasNoIndex, err := backend.Read("/")
	require.NoError(t, err)
	assert.False(t, hasNoIndex)
	require.NotEmpty(t, items)

	// Test reading prefix directory
	items1, hasNoIndex, err := backend.Read("prefix/")
	require.NoError(t, err)
	assert.False(t, hasNoIndex)
	items = append(items, items1...)

	// Test reading subdirectory
	items2, hasNoIndex, err := backend.Read("prefix/dir1/")
	require.NoError(t, err)
	assert.False(t, hasNoIndex)
	items = append(items, items2...)

	require.Len(t, items, 6, "There should be 6 items")

	// Assert the expected items content
	assert.Contains(t, []string{
		"prefix/",
		"file1.txt",
		"file2.txt",
		"smallfile1.txt",
		"dir1/",
	}, items[0].Name)

	// Assert the expected prefixes are "directories"
	for _, item := range items {
		if item.Name == "prefix/" || item.Name == "" || item.Name == "dir1/" || item.Name == "prefix/dir1/" {
			assert.True(t, item.IsDir)
		} else {
			assert.False(t, item.IsDir, "Item %s should not be a directory", item.Name)
		}
	}

	mockSvc.AssertExpectations(t)
}

func TestS3BackendReadWithNoIndex(t *testing.T) {
	mockSvc := new(MockS3Client)
	backend := S3Backend{
		svc:    mockSvc,
		bucket: "test-bucket",
		cfg: Config{
			Recursive:    true,
			Source:       "s3://test-bucket",
			NoIndexFiles: []string{".noindex"},
		},
	}

	// Mock response with a noindex file
	mockSvc.On("ListObjectsV2", mockContext, mock.MatchedBy(func(input *s3.ListObjectsV2Input) bool {
		return *input.Bucket == "test-bucket" && *input.Prefix == "prefix/"
	}), mockOptFns).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key:          aws.String("prefix/.noindex"),
				Size:         aws.Int64(0),
				LastModified: aws.Time(time.Now()),
			},
			{
				Key:          aws.String("prefix/file1.txt"),
				Size:         aws.Int64(1024),
				LastModified: aws.Time(time.Now()),
			},
		},
	}, nil)

	// Test reading directory with noindex file
	t.Logf("NoIndexFiles: %v", backend.cfg.NoIndexFiles)
	items, hasNoIndex, err := backend.Read("prefix/")
	require.NoError(t, err)
	t.Logf("hasNoIndex: %v, items: %v", hasNoIndex, items)
	assert.True(t, hasNoIndex)
	assert.Empty(t, items)

	mockSvc.AssertExpectations(t)
}

func TestS3BackendReadWithNoIndexSimple(t *testing.T) {
	mockSvc := new(MockS3Client)
	backend := S3Backend{
		svc:    mockSvc,
		bucket: "test-bucket",
		cfg: Config{
			NoIndexFiles: []string{".noindex"},
		},
	}

	// Mock response with a noindex file
	mockSvc.On("ListObjectsV2", mockContext, mock.MatchedBy(func(input *s3.ListObjectsV2Input) bool {
		return *input.Bucket == "test-bucket" && (*input.Prefix == "" || *input.Prefix == "/")
	}), mockOptFns).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key:          aws.String(".noindex"),
				Size:         aws.Int64(0),
				LastModified: aws.Time(time.Now()),
			},
		},
	}, nil)

	// Test reading directory with noindex file
	items, hasNoIndex, err := backend.Read("")
	require.NoError(t, err)
	t.Logf("items: %v, hasNoIndex: %v", items, hasNoIndex)
	assert.True(t, hasNoIndex)
	assert.Empty(t, items)

	mockSvc.AssertExpectations(t)
}

func TestS3BackendWrite(t *testing.T) {
	mockSvc := new(MockS3Client)
	s3Backend := S3Backend{
		svc: mockSvc,
		cfg: Config{
			Target:    "s3://test-bucket/",
			BasePath:  "/basepath/",
			IndexFile: "index.html",
		},
	}

	// Setup mock response for PutObject
	mockSvc.On(
		"PutObject", mockContext, mock.AnythingOfType("*s3.PutObjectInput"), mockOptFns,
	).Return(&s3.PutObjectOutput{}, nil)

	data := Data{
		RelativePath: "subdir/",
	}
	content := "<html>Test Content</html>"

	// Execute the Write method
	err := s3Backend.Write(data, content)
	require.NoError(t, err)

	// Verify that PutObject was called as expected
	mockSvc.AssertCalled(t, "PutObject", mockContext, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == "test-bucket" &&
			strings.HasSuffix(*input.Key, "subdir/index.html") &&
			*input.ContentType == "text/html; charset=utf-8"
	}), mockOptFns)

	mockSvc.AssertExpectations(t)
}

func TestIsS3URI(t *testing.T) {
	assert.True(t, isS3URI("s3://test-bucket/"))
	assert.True(t, isS3URI("s3://test-bucket"))
	assert.True(t, isS3URI("s3://test-bucket/one/two/three"))
	assert.False(t, isS3URI("http://example.com/"))
	assert.False(t, isS3URI("/mnt/foo"))
}

func TestS3URIToBucketAndPrefix(t *testing.T) {
	bucket, prefix := uriToBucketAndPrefix("s3://test-bucket/")
	assert.Equal(t, "test-bucket", bucket)
	assert.Equal(t, "", prefix)

	bucket, prefix = uriToBucketAndPrefix("s3://test-bucket")
	assert.Equal(t, "test-bucket", bucket)
	assert.Equal(t, "", prefix)

	bucket, prefix = uriToBucketAndPrefix("s3://test-bucket/one/two/three")
	assert.Equal(t, "test-bucket", bucket)
	assert.Equal(t, "one/two/three", prefix)
}
