// One-off: enforces the new "no H1 in body content" invariant across every
// Tiptap-JSON body column (products.description, services.content,
// branches.description, news.content, projects.description, pages.content).
//
// Every module's rich-text editor used to force the first content block
// into a heading level 1 on every keystroke (elc-tem's normalizeTiptapJson,
// now removed — see elc-tem's shared/lib/tiptap-shared.ts). For
// products/services/branches that first heading was always independent,
// real content — just demote it to level 2. For news/projects/pages the
// title form field was *live-synced from that exact heading's text*
// (elc-tem's now-deleted useTiptapTitleSlugSync) — so it's a literal
// duplicate of the title/name column and gets deleted outright, not
// demoted.
//
// Any other stray level-1 heading found anywhere else in a doc (should
// never exist in practice, since the forced-promotion only ever targeted
// index 0 — this is a defensive pass) is always demoted, never deleted.
//
// Usage: DATABASE_URL=... go run ./cmd/migrate-heading-levels [-commit]
// Dry run by default — prints every row that would change without writing.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type column struct {
	table   string
	jsonCol string
	idCol   string
	// deleteFirst: content[0] being heading level 1 means "delete it"
	// (news/projects/pages, duplicate of the title column) rather than
	// "demote it to level 2" (products/services/branches, real content).
	deleteFirst bool
}

var columns = []column{
	{"products", "description", "id", false},
	{"services", "content", "id", false},
	{"branches", "description", "id", false},
	{"news", "content", "id", true},
	{"projects", "description", "id", true},
	{"pages", "content", "id", true},
}

type rowChange struct {
	id      string
	before  string
	after   string
	payload []byte
}

func main() {
	commit := flag.Bool("commit", false, "actually write changes (default: dry run, print only)")
	flag.Parse()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	var totalChanged int

	for _, c := range columns {
		rows, err := pool.Query(ctx, fmt.Sprintf(
			"SELECT %s, %s FROM %s WHERE %s IS NOT NULL", c.idCol, c.jsonCol, c.table, c.jsonCol))
		if err != nil {
			log.Fatalf("query %s: %v", c.table, err)
		}

		var changes []rowChange
		for rows.Next() {
			var id string
			var raw []byte
			if err := rows.Scan(&id, &raw); err != nil {
				log.Fatalf("scan %s: %v", c.table, err)
			}

			var doc map[string]interface{}
			if err := json.Unmarshal(raw, &doc); err != nil {
				log.Printf("SKIP %s id=%s: invalid json: %v", c.table, id, err)
				continue
			}

			before, after, changed := normalizeDoc(doc, c.deleteFirst)
			if !changed {
				continue
			}

			newRaw, err := json.Marshal(doc)
			if err != nil {
				log.Fatalf("marshal %s id=%s: %v", c.table, id, err)
			}
			changes = append(changes, rowChange{id: id, before: before, after: after, payload: newRaw})
		}
		rows.Close()

		fmt.Printf("=== %s.%s: %d row(s) to change ===\n", c.table, c.jsonCol, len(changes))
		for _, ch := range changes {
			fmt.Printf("  id=%s: %q -> %s\n", ch.id, ch.before, ch.after)
		}
		totalChanged += len(changes)

		if !*commit || len(changes) == 0 {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("begin tx: %v", err)
		}
		for _, ch := range changes {
			if _, err := tx.Exec(ctx,
				fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", c.table, c.jsonCol, c.idCol),
				ch.payload, ch.id); err != nil {
				tx.Rollback(ctx)
				log.Fatalf("update %s id=%s: %v", c.table, ch.id, err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit %s: %v", c.table, err)
		}
		fmt.Printf("  committed %d row(s) for %s.%s\n", len(changes), c.table, c.jsonCol)
	}

	fmt.Printf("\n=== %d row(s) total need changes ===\n", totalChanged)
	if !*commit {
		fmt.Println("Dry run only (pass -commit to write). No changes made.")
	}
}

// normalizeDoc mutates doc["content"] in place (dropping or demoting the
// first node per deleteFirst), then does a defensive demote-only pass over
// every remaining node in the tree. Returns a human-readable before/after
// summary of the first-node change and whether anything changed at all.
func normalizeDoc(doc map[string]interface{}, deleteFirst bool) (before, after string, changed bool) {
	content, ok := doc["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", "", false
	}

	if isLevelOneHeading(content[0]) {
		before = headingText(content[0])
		changed = true
		if deleteFirst {
			content = content[1:]
			after = "(deleted — duplicate of title column)"
		} else {
			demoteHeading(content[0])
			after = "H2: " + headingText(content[0])
		}
	}

	// Defensive: demote any other stray level-1 heading anywhere else in
	// the tree. Never delete — only the confirmed-duplicate first node
	// (handled above) is ever removed.
	for _, node := range content {
		if strayDemote(node) {
			changed = true
		}
	}

	doc["content"] = content
	return before, after, changed
}

func isLevelOneHeading(node interface{}) bool {
	m, ok := node.(map[string]interface{})
	if !ok || m["type"] != "heading" {
		return false
	}
	attrs, _ := m["attrs"].(map[string]interface{})
	level, hasLevel := attrs["level"]
	if !hasLevel {
		return true // missing attrs.level defaults to 1 in legacy docs
	}
	lf, ok := level.(float64)
	return ok && lf == 1
}

func demoteHeading(node interface{}) {
	m, ok := node.(map[string]interface{})
	if !ok {
		return
	}
	attrs, ok := m["attrs"].(map[string]interface{})
	if !ok {
		attrs = map[string]interface{}{}
		m["attrs"] = attrs
	}
	attrs["level"] = float64(2)
}

func strayDemote(node interface{}) bool {
	m, ok := node.(map[string]interface{})
	if !ok {
		return false
	}
	changed := false
	if isLevelOneHeading(node) {
		demoteHeading(node)
		changed = true
	}
	if children, ok := m["content"].([]interface{}); ok {
		for _, child := range children {
			if strayDemote(child) {
				changed = true
			}
		}
	}
	return changed
}

func headingText(node interface{}) string {
	m, ok := node.(map[string]interface{})
	if !ok {
		return ""
	}
	children, ok := m["content"].([]interface{})
	if !ok {
		return ""
	}
	text := ""
	for _, c := range children {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if t, ok := cm["text"].(string); ok {
			text += t
		}
	}
	return text
}
