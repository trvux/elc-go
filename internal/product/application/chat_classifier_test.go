package application

import (
	"context"
	"os"
	"testing"
)

// TestChatClassifierPredict exercises the real fasttext binary + trained
// model via exec.Command, not just the CLI directly — skipped unless both
// are present at the paths given by FASTTEXT_BIN/CHAT_CLASSIFIER_MODEL
// (CI/most dev machines won't have them, same reasoning as
// chatClassifier.available's fallback).
func TestChatClassifierPredict(t *testing.T) {
	bin := os.Getenv("FASTTEXT_BIN")
	model := os.Getenv("CHAT_CLASSIFIER_MODEL")
	if bin == "" || model == "" {
		t.Skip("FASTTEXT_BIN / CHAT_CLASSIFIER_MODEL not set; skipping (see Dockerfile for how these get built)")
	}

	c := &chatClassifier{binPath: bin, modelPath: model, available: true}

	cases := []struct {
		message   string
		wantLabel string
	}{
		{"phòng 20m2 nên dùng máy lạnh nào", "may_lanh"},
		{"tôi cần điều hòa lắp cho văn phòng", "may_lanh"},
		// Neither mentions its category's literal name at all — the whole
		// point of training a model instead of only matching known terms.
		{"nước máy nhà tôi bị đục cần lọc", "loc_nuoc"},
		{"không khí trong nhà hơi bí", "loc_khong_khi"},
	}

	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			label, ok := c.predictCategory(context.Background(), tc.message)
			if !ok {
				t.Fatalf("predictCategory(%q) returned ok=false, want label %q", tc.message, tc.wantLabel)
			}
			if label != tc.wantLabel {
				t.Errorf("predictCategory(%q) = %q, want %q", tc.message, label, tc.wantLabel)
			}
		})
	}
}
