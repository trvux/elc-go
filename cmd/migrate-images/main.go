// One-off migration: copies every image still hosted on Supabase Storage
// (bucket "images") into the R2 bucket, preserving its original folder/file
// path, then rewrites every DB column that stored the old Supabase URL to
// the new R2 URL. Safe to re-run — already-migrated rows (URL already under
// R2_PUBLIC_URL) are skipped.
//
// Two kinds of columns:
//   - textColumns: plain scalar TEXT holding one URL.
//   - jsonColumns: any JSONB column, walked recursively regardless of shape
//     — covers both a flat array of `{"url": ...}` image objects and a
//     deeply nested TipTap/ProseMirror rich-text document with `"src"`
//     image nodes anywhere in the tree. Every string value found anywhere
//     in the JSON is checked against the Supabase Storage URL pattern; only
//     matches are rewritten, everything else (including numbers, via
//     json.Number, to avoid float64 precision loss on round-trip) is left
//     untouched.
//
// Usage: DATABASE_URL=... R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... \
//        R2_SECRET_ACCESS_KEY=... R2_BUCKET_NAME=... R2_PUBLIC_URL=... \
//        go run ./cmd/migrate-images
package main

import (
	"bytes"
	"context"
	"encoding/json"
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

// textColumn is a table/column pair holding a single image URL as plain TEXT.
type textColumn struct {
	table  string
	column string
	idCol  string
}

// jsonColumn is a table/column pair holding JSONB of arbitrary shape that
// may contain Supabase Storage URLs as string values anywhere in the tree.
type jsonColumn struct {
	table  string
	column string
	idCol  string
}

var textColumns = []textColumn{
	{"brands", "logo_url", "id"},
	{"categories", "image_url", "id"},
	{"group_categories", "image_url", "id"},
	{"project_type", "image", "id"},
	{"service_groups", "image_url", "id"},
}

var jsonColumns = []jsonColumn{
	{"branches", "images", "id"},
	{"branches", "description", "id"},
	{"news", "images", "id"},
	{"news", "content", "id"},
	{"services", "images", "id"},
	{"products", "images", "id"},
	{"products", "description", "id"},
	{"projects", "images", "id"},
	{"projects", "description", "id"},
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

	for _, c := range textColumns {
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

	for _, c := range jsonColumns {
		rows, err := pool.Query(ctx, fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s::text LIKE '%%supabase.co%%'", c.idCol, c.column, c.table, c.column))
		if err != nil {
			log.Fatalf("query %s: %v", c.table, err)
		}
		var updates []struct {
			id  string
			raw []byte
		}
		for rows.Next() {
			var id string
			var raw []byte
			if err := rows.Scan(&id, &raw); err != nil {
				log.Fatalf("scan %s: %v", c.table, err)
			}
			updates = append(updates, struct {
				id  string
				raw []byte
			}{id, raw})
		}
		rows.Close()

		for _, u := range updates {
			var parsed any
			dec := json.NewDecoder(bytes.NewReader(u.raw))
			dec.UseNumber()
			if err := dec.Decode(&parsed); err != nil {
				log.Printf("FAILED decode %s.%s id=%s: %v", c.table, c.column, u.id, err)
				continue
			}

			var changed bool
			rewritten := rewriteJSON(parsed, migrateURL, &changed)
			if !changed {
				skipCount++
				continue
			}

			newRaw, err := json.Marshal(rewritten)
			if err != nil {
				log.Printf("FAILED encode %s.%s id=%s: %v", c.table, c.column, u.id, err)
				continue
			}

			_, err = pool.Exec(ctx, fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", c.table, c.column, c.idCol), newRaw, u.id)
			if err != nil {
				log.Printf("FAILED update %s.%s id=%s: %v", c.table, c.column, u.id, err)
			}
		}
	}

	log.Printf("done: %d file(s) copied to R2, %d row(s)/value(s) skipped (already migrated or not a Supabase URL), %d failure(s)", okCount, skipCount, failCount)
}

// rewriteJSON walks an arbitrary decoded JSON value (map[string]any,
// []any, string, json.Number, bool, or nil) and replaces every string that
// migrateURL recognizes as a Supabase Storage URL, regardless of where in
// the tree it appears or what key it's under. Sets *changed to true if any
// replacement happened.
func rewriteJSON(v any, migrateURL func(string) (string, bool), changed *bool) any {
	switch val := v.(type) {
	case string:
		if newURL, ok := migrateURL(val); ok {
			*changed = true
			return newURL
		}
		return val
	case map[string]any:
		for k, vv := range val {
			val[k] = rewriteJSON(vv, migrateURL, changed)
		}
		return val
	case []any:
		for i, vv := range val {
			val[i] = rewriteJSON(vv, migrateURL, changed)
		}
		return val
	default:
		return v // json.Number, bool, nil
	}
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
