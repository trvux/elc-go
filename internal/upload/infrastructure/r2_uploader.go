package infrastructure

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/trvux/elc-go/internal/upload/domain"
)

// R2Uploader stores files in a Cloudflare R2 bucket via its S3-compatible
// API — chosen over Supabase Storage for zero egress fees, and over the
// VPS's own disk so image serving doesn't depend on that one box's uptime
// or storage capacity.
type R2Uploader struct {
	client    *s3.Client
	bucket    string
	publicURL string // base URL prefixed to object keys, no trailing slash (R2 custom domain or public r2.dev bucket URL)
}

var _ domain.Uploader = (*R2Uploader)(nil)

func NewR2Uploader(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) *R2Uploader {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle: true,
	})
	return &R2Uploader{client: client, bucket: bucket, publicURL: publicURL}
}

func (u *R2Uploader) Upload(ctx context.Context, folder, filename, contentType string, content io.Reader) (string, error) {
	key := fmt.Sprintf("%s/%s", folder, filename)
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("r2 uploader: put object: %w", err)
	}
	return fmt.Sprintf("%s/%s", u.publicURL, key), nil
}

// NoopUploader lets the server boot without R2 credentials configured (e.g.
// local dev, or before the bucket/token exist yet) — the route still mounts,
// it just fails clearly instead of the process crashing on missing env vars.
type NoopUploader struct{}

var _ domain.Uploader = (*NoopUploader)(nil)

func NewNoopUploader() *NoopUploader { return &NoopUploader{} }

func (u *NoopUploader) Upload(ctx context.Context, folder, filename, contentType string, content io.Reader) (string, error) {
	return "", fmt.Errorf("image upload is not configured: set R2_ACCOUNT_ID/R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY/R2_BUCKET_NAME/R2_PUBLIC_URL")
}
