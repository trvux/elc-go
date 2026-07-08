package domain

import "strings"

// blockedSubstrings is a small, hardcoded heuristic list — no external API
// (per the plan: auto-filter must work without calling out), covers common
// Vietnamese profanity/spam patterns. Matching is substring, case-insensitive,
// deliberately over-inclusive (false positives just land in the moderation
// queue instead of being rejected — see NewReview).
var blockedSubstrings = []string{
	// Profanity (common Vietnamese slurs/insults)
	"đụ", "địt", "cặc", "lồn", "đéo", "đĩ", "đĩ mẹ", "cc", "vcl", "vl",
	"đm", "đmm", "clm", "dcm", "mẹ mày", "thằng chó", "con chó", "óc chó",
	"ngu như chó", "đồ khốn",
	// Spam/scam patterns
	"http://", "https://", "www.", ".com", ".net", "vay tiền nhanh",
	"lô đề", "cá độ", "casino", "cờ bạc", "khuyến mãi khủng",
	"zalo:", "telegram:",
}

// containsBlockedContent reports whether text contains any blocklisted
// substring, case-insensitively.
func containsBlockedContent(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range blockedSubstrings {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
