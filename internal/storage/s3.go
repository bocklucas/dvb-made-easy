package storage

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Backend struct {
	client *s3.Client
	bucket string
}

func NewS3Backend(creds *S3Creds) (*S3Backend, error) {
	if creds == nil || creds.Bucket == "" {
		return nil, fmt.Errorf("s3 credentials: bucket is required")
	}

	ctx := context.Background()

	// Set up Static Credentials Provider if access/secret key is provided
	var optFns []func(*config.LoadOptions) error
	if creds.AccessKey != "" && creds.SecretKey != "" {
		optFns = append(optFns, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(creds.AccessKey, creds.SecretKey, ""),
		))
	}

	region := creds.Region
	if region == "" {
		region = "us-east-1"
	}
	optFns = append(optFns, config.WithRegion(region))

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("load default config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if creds.Endpoint != "" {
			o.BaseEndpoint = aws.String(creds.Endpoint)
			o.UsePathStyle = true
		}
	})

	return &S3Backend{
		client: client,
		bucket: creds.Bucket,
	}, nil
}

func (b *S3Backend) ListBackups(ctx context.Context, pattern *regexp.Regexp) ([]BackupFile, error) {
	var backups []BackupFile
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            aws.String(b.bucket),
			ContinuationToken: continuationToken,
		}

		resp, err := b.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("s3 list objects: %w", err)
		}

		for _, obj := range resp.Contents {
			key := aws.ToString(obj.Key)
			parts := strings.Split(key, "/")
			filename := parts[len(parts)-1]
			if filename == "" {
				continue
			}

			match, encrypted := MatchBackupPattern(filename, pattern)
			if !match {
				continue
			}

			var lastModified time.Time
			if obj.LastModified != nil {
				lastModified = *obj.LastModified
			}

			backups = append(backups, BackupFile{
				Key:          key,
				Size:         aws.ToInt64(obj.Size),
				LastModified: lastModified,
				IsEncrypted:  encrypted,
			})
		}

		if resp.IsTruncated != nil && !*resp.IsTruncated {
			break
		}
		if resp.NextContinuationToken == nil {
			break
		}
		continuationToken = resp.NextContinuationToken
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Key > backups[j].Key
	})

	return backups, nil
}

func (b *S3Backend) Download(ctx context.Context, key string, w io.Writer) error {
	input := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	resp, err := b.client.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("s3 get object %s: %w", key, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("copy s3 body %s: %w", key, err)
	}

	return nil
}

func (b *S3Backend) TestConnection(ctx context.Context) error {
	_, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.bucket),
	})
	if err != nil {
		return fmt.Errorf("s3 head bucket %s: %w", b.bucket, err)
	}
	return nil
}
