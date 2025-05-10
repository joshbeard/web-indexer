package webindexer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/charmbracelet/log"
)

type S3Backend struct {
	svc    S3API
	bucket string
	cfg    Config
}

type S3API interface {
	ListObjectsV2(input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

var _ FileSource = &S3Backend{}

func (s *S3Backend) Read(prefix string) ([]*Item, bool, error) {
	// Ensure the prefix has a trailing slash for s3 keys
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// Remove leading slash for s3 keys
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix == "/" {
		prefix = ""
	}

	log.Debugf("Listing objects in %s/%s", s.bucket, prefix)

	req := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	resp, err := s.svc.ListObjectsV2(req)
	if err != nil {
		return nil, false, fmt.Errorf("unable to list S3 objects: %w", err)
	}

	// First check for noindex files or skipindex files before processing anything else
	for _, content := range resp.Contents {
		fileName := filepath.Base(*content.Key)
		// Check for noindex files (skip directory entirely)
		if len(s.cfg.NoIndexFiles) > 0 && contains(s.cfg.NoIndexFiles, fileName) {
			log.Infof("Skipping %s/%s (found noindex file %s)", s.bucket, prefix, fileName)
			return nil, true, nil
		}

		// Check for skipindex files (skip indexing but include in parent)
		if len(s.cfg.SkipIndexFiles) > 0 && contains(s.cfg.SkipIndexFiles, fileName) {
			log.Infof("Skipping indexing of %s/%s (found skipindex file %s), will include in parent directory", s.bucket, prefix, fileName)
			// Return empty items but mark as not having noindex file
			// This will prevent indexing this directory but still include it in the parent
			return []*Item{}, false, nil
		}
	}

	var items []*Item
	// Process all other files
	for _, content := range resp.Contents {
		if shouldSkip(*content.Key, s.cfg.IndexFile, s.cfg.Skips) {
			continue
		}

		// Get the relative name by removing the prefix
		itemName := strings.TrimPrefix(*content.Key, prefix)

		item := &Item{
			Name:         itemName,
			Size:         *content.Size,
			LastModified: *content.LastModified,
			IsDir:        false,
			HasMetadata:  true,
		}

		items = append(items, item)
	}

	// Only process directories if we haven't found a noindex file
	for _, commonPrefix := range resp.CommonPrefixes {
		log.Debugf("Found common prefix: %s", *commonPrefix.Prefix)

		// Check if this prefix contains a noindex file
		subReq := &s3.ListObjectsV2Input{
			Bucket:    aws.String(s.bucket),
			Prefix:    commonPrefix.Prefix,
			Delimiter: aws.String("/"),
		}

		subResp, err := s.svc.ListObjectsV2(subReq)
		if err != nil {
			return nil, false, fmt.Errorf("unable to list S3 objects in prefix %s: %w", *commonPrefix.Prefix, err)
		}

		// Skip this prefix if it contains a noindex file
		skipDir := false
		for _, content := range subResp.Contents {
			fileName := filepath.Base(*content.Key)
			if len(s.cfg.NoIndexFiles) > 0 && contains(s.cfg.NoIndexFiles, fileName) {
				log.Infof("Skipping %s/%s (found noindex file %s)", s.bucket, *commonPrefix.Prefix, fileName)
				skipDir = true
				break
			}
		}
		if skipDir {
			continue
		}

		dirName := strings.TrimPrefix(*commonPrefix.Prefix, prefix)
		item := &Item{
			Name:  dirName,
			IsDir: true,
		}
		items = append(items, item)
	}

	return items, false, nil
}

// EnsureDirExists is a no-op for S3 as directories are implicit.
func (s *S3Backend) EnsureDirExists(relativePath string) error {
	log.Debugf("EnsureDirExists called for S3 (no-op): %s/%s", s.bucket, relativePath)
	// S3 directories are created implicitly by object keys.
	// We could potentially check if the bucket exists here if needed.
	return nil
}

func (s *S3Backend) Write(data Data, content string) error {
	bucket, target := uriToBucketAndPrefix(s.cfg.Target)
	target = strings.TrimPrefix(target, s.cfg.BasePath)
	target = filepath.Join(target, data.RelativePath, s.cfg.IndexFile)

	strReader := strings.NewReader(content)
	size := humanizeBytes(int64(strReader.Len()))
	log.Infof("Uploading %s to %s/%s", size, bucket, target)

	_, err := s.svc.PutObject(&s3.PutObjectInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(target),
		Body:            aws.ReadSeekCloser(strReader),
		ContentType:     aws.String("text/html"),
		ContentEncoding: aws.String("utf-8"),
	})
	return err
}

func isS3URI(uri string) bool {
	return strings.HasPrefix(uri, "s3://")
}

func uriToBucketAndPrefix(uri string) (string, string) {
	uri = strings.TrimPrefix(uri, "s3://")
	uriParts := strings.SplitN(uri, "/", 2)

	if len(uriParts) == 1 {
		return uriParts[0], ""
	}

	return uriParts[0], uriParts[1]
}
