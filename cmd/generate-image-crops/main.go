// One-off backfill: generates the 4:3/1:1 crop variants (see
// internal/upload's cropVariantsFolder feature) for every product image
// that predates that feature — new uploads already get these at upload
// time, this just catches up the existing catalog. Safe to re-run: any
// image that already has cropVariants is skipped.
//
// Requires the `cwebp` binary on PATH (apk/brew package libwebp/
// libwebp-tools) — this script isn't run inside the server's Docker image
// (which ships a compiled binary only, no Go toolchain to `go run` with),
// so unlike production it needs cwebp installed on whatever machine runs it.
//
// Usage: DATABASE_URL=... R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... \
//        R2_SECRET_ACCESS_KEY=... R2_BUCKET_NAME=... R2_PUBLIC_URL=... \
//        go run ./cmd/generate-image-crops
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/upload/infrastructure"
)

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

	uploader := infrastructure.NewR2Uploader(r2AccountID, r2AccessKeyID, r2SecretAccessKey, r2Bucket, r2PublicURL)
	cropper := infrastructure.NewImageCropper()
	httpClient := http.Client{Timeout: 30 * time.Second}

	rows, err := pool.Query(ctx, `
		SELECT id, images FROM products
		WHERE deleted_at IS NULL AND images IS NOT NULL AND images != '[]'::jsonb`)
	if err != nil {
		log.Fatalf("query products: %v", err)
	}
	type productRow struct {
		id        string
		imagesRaw []byte
	}
	var products []productRow
	for rows.Next() {
		var p productRow
		if err := rows.Scan(&p.id, &p.imagesRaw); err != nil {
			log.Fatalf("scan product: %v", err)
		}
		products = append(products, p)
	}
	rows.Close()

	log.Printf("found %d product(s) with images", len(products))

	var okCount, skipCount, failCount int
	for _, p := range products {
		images, err := media.UnmarshalImages(p.imagesRaw)
		if err != nil {
			log.Printf("FAILED product %s: unmarshal images: %v", p.id, err)
			failCount++
			continue
		}

		changed := false
		for i, img := range images {
			if len(img.CropVariants) > 0 {
				skipCount++
				continue
			}

			resp, err := httpClient.Get(img.URL)
			if err != nil {
				log.Printf("FAILED product %s image %s: download: %v", p.id, img.URL, err)
				failCount++
				continue
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				log.Printf("FAILED product %s image %s: download status %d", p.id, img.URL, resp.StatusCode)
				failCount++
				continue
			}

			crops, err := cropper.GenerateCropVariants(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Printf("FAILED product %s image %s: generate crops: %v", p.id, img.URL, err)
				failCount++
				continue
			}

			base := strings.TrimSuffix(path.Base(img.URL), path.Ext(img.URL))
			variants := make(map[string]string, len(crops))
			for label, data := range crops {
				filename := fmt.Sprintf("%s-%s.webp", base, strings.ReplaceAll(label, ":", "x"))
				url, err := uploader.Upload(ctx, "products", filename, "image/webp", bytes.NewReader(data))
				if err != nil {
					log.Printf("FAILED product %s image %s: upload %s crop: %v", p.id, img.URL, label, err)
					continue
				}
				variants[label] = url
			}
			if len(variants) == 0 {
				failCount++
				continue
			}

			images[i].CropVariants = variants
			changed = true
			okCount++
			if okCount%25 == 0 {
				log.Printf("progress: %d image(s) processed so far", okCount)
			}
		}

		if !changed {
			continue
		}

		newRaw, err := media.MarshalImages(images)
		if err != nil {
			log.Printf("FAILED product %s: marshal images: %v", p.id, err)
			failCount++
			continue
		}
		if _, err := pool.Exec(ctx, "UPDATE products SET images = $1 WHERE id = $2", newRaw, p.id); err != nil {
			log.Printf("FAILED product %s: update: %v", p.id, err)
			failCount++
		}
	}

	log.Printf("done: %d image(s) cropped, %d skipped (already had crops), %d failure(s)", okCount, skipCount, failCount)
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s must be set", name)
	}
	return v
}
