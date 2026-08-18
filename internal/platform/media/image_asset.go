// Package media holds the shared ImageAsset value object used by every
// module that stores a gallery of images as jsonb (catalog, project,
// service, branch, news) — a plain []string of URLs has no room for alt
// text/caption, which schema.org's ImageObject (and every major CMS/e-
// commerce platform: WordPress Media, Shopify's {src, altText}) models as a
// structured object, not a bare string. Centralized here rather than
// duplicated per module (unlike Seo/CategoryRef, which are intentionally
// duplicated per module to avoid a cross-module domain dependency) because
// this is infrastructure-shaped code — marshal/unmarshal mechanics, not
// business logic — the same reasoning internal/platform/apperr and
// internal/platform/db are shared.
package media

import "encoding/json"

type ImageAsset struct {
	URL     string `json:"url"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
	// CropVariants maps aspect ratio label ("4:3", "1:1") to a derivative
	// crop of URL, generated at upload time (see internal/upload) — lets
	// Product JSON-LD offer Google multiple image aspect ratios without
	// storing/serving anything beyond what one upload already produces.
	// Empty/absent for images uploaded before this existed.
	CropVariants map[string]string `json:"cropVariants,omitempty"`
}

// MarshalImages/UnmarshalImages hand-roll the jsonb <-> []ImageAsset
// conversion — every module's shared pool runs pgx.QueryExecModeSimpleProtocol
// (PgBouncer fix), which can't infer an OID for an arbitrary struct slice;
// the marshaled result must be json.RawMessage specifically (not a bare
// []byte — pgx encodes a bare []byte as bytea, which Postgres then rejects
// casting to jsonb). Same pattern as catalog's marshalSpecs/brand's
// marshalFAQ, just centralized since every module needs the identical logic.
func MarshalImages(images []ImageAsset) (json.RawMessage, error) {
	if images == nil {
		images = []ImageAsset{}
	}
	return json.Marshal(images)
}

func UnmarshalImages(raw []byte) ([]ImageAsset, error) {
	if len(raw) == 0 {
		return []ImageAsset{}, nil
	}
	var images []ImageAsset
	if err := json.Unmarshal(raw, &images); err != nil {
		return nil, err
	}
	if images == nil {
		images = []ImageAsset{}
	}
	return images, nil
}

// FirstURL returns the primary/cover image URL, or "" if there are none —
// the common case every public detail/card/OG-image/JSON-LD consumer needs.
func FirstURL(images []ImageAsset) string {
	if len(images) == 0 {
		return ""
	}
	return images[0].URL
}
