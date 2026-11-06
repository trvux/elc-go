package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/trvux/elc-go/internal/upload/domain"
)

// R2FileStore keeps customer files in a PRIVATE Cloudflare R2 bucket. The bucket has no
// public URL; staff read a file through a signed link that expires in minutes.
type R2FileStore struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

var _ domain.FileStore = (*R2FileStore)(nil)

// NewR2FileStore connects to bucket on the account's S3-compatible endpoint, or to
// endpointOverride when set (a local MinIO in tests).
func NewR2FileStore(accountID, accessKeyID, secretAccessKey, bucket, endpointOverride string) *R2FileStore {
	endpoint := endpointOverride
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle: true,
	})
	return &R2FileStore{client: client, presign: s3.NewPresignClient(client), bucket: bucket}
}

func (s *R2FileStore) Put(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("r2 filestore: put object: %w", err)
	}
	return nil
}

func (s *R2FileStore) Stat(ctx context.Context, key string) (domain.FileInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		var notFound *types.NotFound
		var apiErr smithy.APIError
		if errors.As(err, &notFound) || (errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey")) {
			return domain.FileInfo{}, domain.ErrFileNotFound
		}
		return domain.FileInfo{}, fmt.Errorf("r2 filestore: head object: %w", err)
	}
	return domain.FileInfo{Size: aws.ToInt64(out.ContentLength), ContentType: aws.ToString(out.ContentType)}, nil
}

func (s *R2FileStore) SignedURL(ctx context.Context, key string, ttl time.Duration, name, contentType string, inline bool) (string, error) {
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		// mime.FormatMediaType quotes and RFC 2231-encodes the name, so a file called
		// `a"; b=c.pdf` cannot inject a header parameter.
		ResponseContentDisposition: aws.String(mime.FormatMediaType(disposition, map[string]string{"filename": name})),
	}
	if contentType != "" {
		input.ResponseContentType = aws.String(contentType)
	}
	req, err := s.presign.PresignGetObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("r2 filestore: presign: %w", err)
	}
	return req.URL, nil
}

func (s *R2FileStore) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("r2 filestore: delete object: %w", err)
	}
	return nil
}

// NoopFileStore lets the server boot without the private bucket configured: file
// uploads fail with a clear message instead of the process crashing on missing env vars.
type NoopFileStore struct{}

var _ domain.FileStore = (*NoopFileStore)(nil)

func NewNoopFileStore() *NoopFileStore { return &NoopFileStore{} }

var errFileStoreNotConfigured = errors.New("file storage is not configured: set R2_PRIVATE_BUCKET_NAME/R2_PRIVATE_ACCESS_KEY_ID/R2_PRIVATE_SECRET_ACCESS_KEY")

func (NoopFileStore) Put(context.Context, string, string, io.ReadSeeker) error {
	return errFileStoreNotConfigured
}
func (NoopFileStore) Stat(context.Context, string) (domain.FileInfo, error) {
	return domain.FileInfo{}, errFileStoreNotConfigured
}
func (NoopFileStore) SignedURL(context.Context, string, time.Duration, string, string, bool) (string, error) {
	return "", errFileStoreNotConfigured
}
func (NoopFileStore) Delete(context.Context, string) error { return errFileStoreNotConfigured }
