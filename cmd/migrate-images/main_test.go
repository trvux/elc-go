package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestRewriteJSON_ImageArray(t *testing.T) {
	// Shape observed on branches.images / news.images / services.images /
	// products.images / projects.images: flat JSONB array of image objects.
	raw := []byte(`[
		{"url": "https://gdzihzsjfczuggwpykjk.supabase.co/storage/v1/object/public/images/branches/a.webp", "alt": null},
		{"url": "https://media.dienmayelc.com.vn/already/migrated.webp", "alt": "kept"}
	]`)

	migrateURL := func(u string) (string, bool) {
		if u == "https://gdzihzsjfczuggwpykjk.supabase.co/storage/v1/object/public/images/branches/a.webp" {
			return "https://pub-xyz.r2.dev/branches/a.webp", true
		}
		return u, false
	}

	var parsed any
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var changed bool
	result := rewriteJSON(parsed, migrateURL, &changed)
	if !changed {
		t.Fatal("expected changed=true")
	}

	out, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if got[0]["url"] != "https://pub-xyz.r2.dev/branches/a.webp" {
		t.Errorf("url not rewritten: %v", got[0]["url"])
	}
	if got[0]["alt"] != nil {
		t.Errorf("alt should stay null, got %v", got[0]["alt"])
	}
	if got[1]["url"] != "https://media.dienmayelc.com.vn/already/migrated.webp" {
		t.Errorf("already-migrated url should be untouched, got %v", got[1]["url"])
	}
	if got[1]["alt"] != "kept" {
		t.Errorf("unrelated field mutated: %v", got[1]["alt"])
	}
}

func TestRewriteJSON_NestedTipTapDoc(t *testing.T) {
	// Shape observed on projects.description / news.content /
	// branches.description / products.description: arbitrarily nested
	// ProseMirror/TipTap document, image node buried inside "content" arrays.
	raw := []byte(`{
		"type": "doc",
		"content": [
			{"type": "paragraph", "content": [{"type": "text", "text": "hello"}]},
			{"type": "image", "attrs": {"alt": null, "src": "https://gdzihzsjfczuggwpykjk.supabase.co/storage/v1/object/public/images/projects/x.webp", "align": "center", "ratio": "auto", "title": null, "width": 800}}
		]
	}`)

	migrateURL := func(u string) (string, bool) {
		if u == "https://gdzihzsjfczuggwpykjk.supabase.co/storage/v1/object/public/images/projects/x.webp" {
			return "https://pub-xyz.r2.dev/projects/x.webp", true
		}
		return u, false
	}

	var parsed any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var changed bool
	result := rewriteJSON(parsed, migrateURL, &changed)
	if !changed {
		t.Fatal("expected changed=true")
	}

	out, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	outDec := json.NewDecoder(bytes.NewReader(out))
	outDec.UseNumber()
	if err := outDec.Decode(&got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	content := got["content"].([]any)
	imgNode := content[1].(map[string]any)
	attrs := imgNode["attrs"].(map[string]any)

	if attrs["src"] != "https://pub-xyz.r2.dev/projects/x.webp" {
		t.Errorf("src not rewritten: %v", attrs["src"])
	}
	// width was encoded as a JSON number (800) — must survive the
	// UseNumber() round-trip without turning into 800.0 or losing precision.
	if w, ok := attrs["width"].(json.Number); !ok || w.String() != "800" {
		t.Errorf("width corrupted by round-trip: %#v", attrs["width"])
	}

	// paragraph/text branch of the tree must be untouched.
	para := content[0].(map[string]any)
	textNode := para["content"].([]any)[0].(map[string]any)
	if textNode["text"] != "hello" {
		t.Errorf("unrelated text node mutated: %v", textNode["text"])
	}
}

func TestRewriteJSON_NoMatchLeavesValueUnchanged(t *testing.T) {
	raw := []byte(`{"a": "plain string", "b": [1, 2, 3], "c": null, "d": true}`)

	migrateURL := func(u string) (string, bool) { return u, false }

	var parsed any
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	original := parsed

	var changed bool
	result := rewriteJSON(parsed, migrateURL, &changed)
	if changed {
		t.Fatal("expected changed=false when nothing matches")
	}
	if !reflect.DeepEqual(original, result) {
		t.Errorf("value should be structurally identical: got %#v", result)
	}
}
