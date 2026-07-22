package application

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// chatClassifier wraps a locally-run fastText model — no LLM, no
// embeddings, no per-call API cost, just a ~330KB quantized model file and
// the fasttext CLI binary invoked via exec.Command (there's no pure-Go
// fastText inference library, and shelling out to the compiled binary
// avoids taking on CGO for the rest of this otherwise CGO_ENABLED=0
// build — see Dockerfile). Trained offline from cmd/train-chat-classifier
// on synthetic examples generated from a category/synonym lookup table.
type chatClassifier struct {
	binPath   string
	modelPath string
	available bool
}

var defaultChatClassifier = newChatClassifier()

func newChatClassifier() *chatClassifier {
	bin := envOr("FASTTEXT_BIN", "/usr/local/bin/fasttext")
	model := envOr("CHAT_CLASSIFIER_MODEL", "/app/chat-classifier.ftz")
	_, binErr := os.Stat(bin)
	_, modelErr := os.Stat(model)
	return &chatClassifier{
		binPath:   bin,
		modelPath: model,
		// Both missing in local dev (nobody's expected to have the
		// fasttext binary installed just to `go run` the server) — the
		// caller falls back to the regex/synonym table whenever this is
		// false, so that's not an error, just a quieter code path.
		available: binErr == nil && modelErr == nil,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// chatClassifierMinConfidence: below this, the prediction is discarded
// rather than trusted — better to fall back to the regex synonym table on
// an ambiguous message than confidently act on a wrong guess. 0.5 (not
// higher) because longer narrative messages the training data doesn't
// have close templates for ("Phòng 15m2 nhưng tường gạch mỏng bị nắng
// chiếu...") measured 0.58-0.60 in practice despite being correctly
// classified — with only 4 labels (random-guess baseline 0.25), 0.5 is
// still 2x that baseline, not a coin flip.
const chatClassifierMinConfidence = 0.5

// chatClassifierTimeout bounds a single prediction — the compiled binary
// on a ~330KB model is normally sub-10ms, this is just a safety net so a
// stuck/missing binary can never hang a chat-search request.
const chatClassifierTimeout = 500 * time.Millisecond

// predictCategory returns the predicted category label (a key of
// categoryLabelCanonical, e.g. "may_lanh") and true if the classifier is
// available and confident; otherwise ("", false), signaling the caller to
// fall back to categorySynonyms.
func (c *chatClassifier) predictCategory(ctx context.Context, text string) (string, bool) {
	if !c.available || strings.TrimSpace(text) == "" {
		return "", false
	}

	ctx, cancel := context.WithTimeout(ctx, chatClassifierTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.binPath, "predict-prob", c.modelPath, "-")
	// fastText treats each line of stdin as one document to classify —
	// a stray newline in the message would be read as two documents, so
	// collapse it defensively even though chat input is normally one line.
	cmd.Stdin = strings.NewReader(strings.ReplaceAll(text, "\n", " ") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	if !scanner.Scan() {
		return "", false
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) != 2 {
		return "", false
	}
	label := strings.TrimPrefix(fields[0], "__label__")
	confidence, err := strconv.ParseFloat(fields[1], 64)
	if err != nil || confidence < chatClassifierMinConfidence {
		return "", false
	}
	return label, true
}
