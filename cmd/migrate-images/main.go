// One-off migration: copies every image still hosted on Supabase Storage
// (bucket "images") into the R2 bucket, preserving its original folder/file
// path, then rewrites every DB column that stored the old Supabase URL to
// the new R2 URL. Safe to re-run — already-migrated rows (URL already under
// R2_PUBLIC_URL) are skipped.
//
// Usage: DATABASE_URL=... R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... \
//        R2_SECRET_ACCESS_KEY=... R2_BUCKET_NAME=... R2_PUBLIC_URL=... \
//        go run ./cmd/migrate-images
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// scalarColumn is a table/column pair holding a single image URL.
type scalarColumn struct {
	table  string
	column string
	idCol  string
}

// arrayColumn is a table/column pair holding a text[] of image URLs.
type arrayColumn struct {
	table  string
	column string
	idCol  string
}

var scalarColumns = []scalarColumn{
	{"branches", "image_url", "id"},
	{"brands", "logo_url", "id"},
	{"categories", "image_url", "id"},
	{"group_categories", "image_url", "id"},
	{"news", "image", "id"},
	{"old_services", "image", "id"},
	{"project_type", "image", "id"},
	{"service_groups", "image_url", "id"},
	{"services", "image", "id"},
}

var arrayColumns = []arrayColumn{
	{"products", "images", "id"},
	{"projects", "images", "id"},
}

var supabaseImagePathRe = regexp.MustCompile(`https://[^/]+\.supabase\.co/storage/v1/object/public/images/(.+)$`)

func main() {
	ctx := context.Background()

	databaseURL := requireEnv("DATABASE_URL")
	r2AccountID := requireEnv("R2_ACCOUNT_ID")
	r2AccessKeyID := requireEnv("R2_ACCESS_KEY_ID")
	r2SecretAccessKey := requireEnv("R2_SECRET_ACCESS_KEY")
	r2Bucket := requireEnv("R2_BUCKET_NAME")
	r2PublicURL := strings.TrimRight(requireEnv("R2_PUBLIC_URL"), "/")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	s3Client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2AccountID)),
		Credentials:  credentials.NewStaticCredentialsProvider(r2AccessKeyID, r2SecretAccessKey, ""),
		UsePathStyle: true,
	})

	migrated := map[string]string{} // old url -> new url, reused across all tables
	var okCount, failCount, skipCount int

	migrateURL := func(oldURL string) (string, bool) {
		if strings.HasPrefix(oldURL, r2PublicURL) {
			return oldURL, false // already migrated
		}
		if newURL, ok := migrated[oldURL]; ok {
			return newURL, true
		}
		m := supabaseImagePathRe.FindStringSubmatch(oldURL)
		if m == nil {
			return oldURL, false // not a Supabase Storage URL we recognize — leave alone
		}
		key := m[1]

		newURL, err := copyToR2(ctx, s3Client, r2Bucket, r2PublicURL, oldURL, key)
		if err != nil {
			log.Printf("FAILED %s: %v", oldURL, err)
			failCount++
			return oldURL, false
		}
		migrated[oldURL] = newURL
		okCount++
		if okCount%25 == 0 {
			log.Printf("progress: %d file(s) copied so far", okCount)
		}
		return newURL, true
	}

	for _, c := range scalarColumns {
		rows, err := pool.Query(ctx, fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s LIKE '%%supabase.co%%'", c.idCol, c.column, c.table, c.column))
		if err != nil {
			log.Fatalf("query %s: %v", c.table, err)
		}
		var updates []struct {
			id  string
			url string
		}
		for rows.Next() {
			var id, url string
			if err := rows.Scan(&id, &url); err != nil {
				log.Fatalf("scan %s: %v", c.table, err)
			}
			updates = append(updates, struct {
				id  string
				url string
			}{id, url})
		}
		rows.Close()

		for _, u := range updates {
			newURL, changed := migrateURL(u.url)
			if !changed {
				skipCount++
				continue
			}
			_, err := pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", c.table, c.column, c.idCol), newURL, u.id)
			if err != nil {
				log.Printf("FAILED update %s.%s id=%s: %v", c.table, c.column, u.id, err)
			}
		}
	}

	for _, c := range arrayColumns {
		rows, err := pool.Query(ctx, fmt.Sprintf(
			"SELECT %s, %s FROM %s WHERE EXISTS (SELECT 1 FROM unnest(%s) u WHERE u LIKE '%%supabase.co%%')",
			c.idCol, c.column, c.table, c.column))
		if err != nil {
			log.Fatalf("query %s: %v", c.table, err)
		}
		var updates []struct {
			id   string
			urls []string
		}
		for rows.Next() {
			var id string
			var urls []string
			if err := rows.Scan(&id, &urls); err != nil {
				log.Fatalf("scan %s: %v", c.table, err)
			}
			updates = append(updates, struct {
				id   string
				urls []string
			}{id, urls})
		}
		rows.Close()

		for _, u := range updates {
			newURLs := make([]string, len(u.urls))
			anyChanged := false
			for i, url := range u.urls {
				newURL, changed := migrateURL(url)
				newURLs[i] = newURL
				if changed {
					anyChanged = true
				}
			}
			if !anyChanged {
				skipCount++
				continue
			}
			_, err := pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", c.table, c.column, c.idCol), newURLs, u.id)
			if err != nil {
				log.Printf("FAILED update %s.%s id=%s: %v", c.table, c.column, u.id, err)
			}
		}
	}

	log.Printf("done: %d file(s) copied to R2, %d row(s)/value(s) skipped (already migrated or not a Supabase URL), %d failure(s)", okCount, skipCount, failCount)
}

func copyToR2(ctx context.Context, client *s3.Client, bucket, publicURL, sourceURL, key string) (string, error) {
	httpClient := http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(sourceURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(string(body)),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	return publicURL + "/" + key, nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s must be set", name)
	}
	return v
}
