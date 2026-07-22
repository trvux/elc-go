package application

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/trvux/elc-go/internal/product/domain"
)

// ChatSearchLimit caps how many products a single chat-style query returns
// — this is meant to read as "here are a few good matches", not another
// paginated grid.
const ChatSearchLimit = 6

// ChatSearchProducts finds a short, curated list of products matching a
// free-text shopper message — the conversational counterpart to the full
// filtered grid, for when someone would rather describe what they need
// ("máy lạnh dưới 10 triệu cho phòng nhỏ") than click through filters.
//
// No LLM, no per-call API cost: category detection tries a small,
// locally-run fastText model first (trained offline from a lookup table —
// see cmd/train-chat-classifier — so it generalizes to phrasing like
// "nước nhà tôi bị đục cần lọc" that never mentions "lọc nước" at all),
// falling back to a hand-written regex/synonym table wherever the model
// isn't available (e.g. local dev without it built) or isn't confident.
// Price-range extraction and room-area-to-HP mapping are plain rules
// either way — those are deterministic formulas, not a language
// understanding problem, so a model doesn't do them any better. All of it
// feeds the existing faceted domain.ProductFilter, backed entirely by the
// search_vector/trigram full-text search Postgres already runs for the
// regular listing page (see ProductFilter.Search).
// ChatSearchResult wraps the regular product-list result with a chat-
// search-specific Explanation (see buildExplanation) — a plain-language,
// template-assembled (not LLM-generated) sentence describing why these
// particular filters were chosen, so a shopper isn't just handed a bare
// product list with no visible reasoning. Kept separate from
// domain.ProductListResult since that type is shared with the plain
// listing endpoint, which has no concept of this.
type ChatSearchResult struct {
	*domain.ProductListResult
	Explanation string
	// Suggestions are follow-up queries a shopper can tap to dig deeper
	// from wherever this turn's answer left off — see buildSuggestions.
	// Self-contained query strings (not questions needing a reply), since
	// the Go side is stateless per request: clicking one just becomes the
	// next turn's `message`, and buildSuggestions already folded this
	// turn's resolved context (form factor, capacity, ...) into its text,
	// so no conversation history needs threading through the API for the
	// next round to keep digging deeper.
	Suggestions []string
}

func ChatSearchProducts(ctx context.Context, repo domain.ProductRepository, message string) (*ChatSearchResult, error) {
	filter, narrowSearch, explanation, signals := parseChatQuery(ctx, message)
	result, err := repo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Progressive relaxation: the classifier-confident path folds
	// marketing/feature words into Search (see parseChatQuery) so a query
	// like "máy lạnh chống khô da" can match against description/highlights
	// text (migration 000018), not just the product name. But
	// websearch_to_tsquery ANDs every word, so a feature phrase that
	// doesn't happen to appear verbatim in any matching product's indexed
	// text would zero out results the shopper's actual category/brand/
	// price/attribute filters would otherwise satisfy. Retrying once with
	// just the narrow (brand-only, or no text at all) search beats
	// returning nothing over one AND-clause miss — narrowSearch=="" is a
	// legitimate, useful fallback value here (not "nothing to retry
	// with"): CategorySlugs/AttributeTokens/price already carry the real
	// correctness guarantee (see parseChatQuery), so dropping Search
	// entirely and relying on those alone is a safe, sensible relaxation
	// step, not a giveaway.
	if result.TotalCount == 0 && narrowSearch != filter.Search {
		filter.Search = narrowSearch
		result, err = repo.GetAll(ctx, filter)
		if err != nil {
			return nil, err
		}
	}
	return &ChatSearchResult{
		ProductListResult: result,
		Explanation:       explanation,
		Suggestions:       buildSuggestions(signals, result.TotalCount),
	}, nil
}

// chatQuerySignals carries the same parsed-context pieces buildExplanation
// uses, but kept around after parseChatQuery returns so ChatSearchProducts
// can also feed them (plus the search's actual TotalCount, not available
// until after repo.GetAll runs) into buildSuggestions.
type chatQuerySignals struct {
	label            string // classifier label; "" on the fallback path (no confident category)
	canonical        string // canonical category term, e.g. "máy lạnh"; "" alongside label
	hasSub           bool
	sub              string
	capacity         capacityInfo
	gas              string
	buildingTypeName string
	minPrice         *int64
	maxPrice         *int64
	sortBy           string
}

// parseChatQuery returns the filter to search with, a narrower fallback
// Search string for ChatSearchProducts' progressive-relaxation retry (see
// its doc comment), a client-facing explanation (see buildExplanation) of
// why these filters were chosen, and the raw signals buildSuggestions uses
// to propose follow-ups.
func parseChatQuery(ctx context.Context, message string) (domain.ProductFilter, string, string, chatQuerySignals) {
	status := domain.ProductStatusPublished
	filter := domain.ProductFilter{
		Status: &status,
		Limit:  ChatSearchLimit,
	}

	// Try the trained classifier on the raw message first — it can infer
	// category from context words alone (see doc comment above), which
	// the regex/synonym table below never can. When confident, the
	// canonical category term anchors the Search text — plus a known
	// brand and/or sub-category (form factor) if mentioned, since (unlike
	// generic filler) those are looked up against real lists, not guessed,
	// and they *do* appear in product names.
	//
	// Leftover content words (after structured signals and stopwords are
	// stripped) are appended too: search_vector now also indexes
	// description/highlights text (migration 000018), so a feature phrase
	// like "chống khô da" or "tiết kiệm điện" can genuinely match a
	// product's marketing copy even though it never appears in the name.
	// This does make the query stricter (AND across more words), which is
	// exactly what the caller's progressive-relaxation retry (see
	// ChatSearchProducts) exists to undo if it overshoots.
	if label, ok := defaultChatClassifier.predictCategory(ctx, message); ok {
		if canonical, ok := categoryLabelCanonical[label]; ok {
			remaining, capacity, gas := applyStructuredSignals(&filter, message)
			// subCategoryTerms is AC form-factor vocabulary ("âm trần",
			// "treo tường", ...) — only meaningful for label == "may_lanh".
			// Checking it for other categories risks a false hit: "âm
			// trần" also means plain "ceiling-mounted" in general
			// construction talk (e.g. "thi công âm trần" for a fresh-air
			// system installation), which isn't the AC form factor at all.
			var sub string
			var hasSub bool
			if label == "may_lanh" {
				sub, hasSub = detectSubCategory(message)
			}
			applyCategorySlugs(&filter, label, sub, message, hasSub)

			// Independent of capacity.buildingTypeName (only set when a
			// building type drove the *heat-load* adjustment, which needs
			// an area to adjust) — a building type can also be the reason
			// the *category* was chosen with no area mentioned at all
			// ("biệt thự nên lắp máy lạnh Daikin nào"), and the
			// explanation should say so either way.
			var buildingTypeName string
			if bt, ok := detectBuildingType(message); ok {
				buildingTypeName = bt.name
			}

			// Neither the canonical category term nor the sub-category
			// term goes into Search anymore — CategorySlugs (just above)
			// is now a hard, DB-level guarantee of both, which makes
			// requiring them as a mandatory AND/phrase in Search redundant
			// at best. At worst actively harmful: verified against real
			// data that zero of Acis's 16 nhà-thông-minh products contain
			// the literal phrase "nhà thông minh" anywhere, and only 4 of
			// 59 treo-tường products literally say "treo tường" in their
			// name/description (it's the unmarked default wall-mount
			// type, so most product copy never bothers stating it) — both
			// made their respective category/sub-category queries return
			// 0 despite CategorySlugs already having the right answer.
			// Search now only needs to *discriminate within* the
			// category (brand/feature words), not re-prove it.
			var terms []string
			brand, hasBrand := detectBrand(message)
			if hasBrand {
				terms = append(terms, brand)
			}
			narrowSearch := quoteKnownPhrases(strings.Join(terms, " "))

			for _, syn := range categorySynonyms {
				remaining = syn.pattern.ReplaceAllString(remaining, "")
			}
			// Strip the canonical/brand/sub-category words out of the
			// leftover text too, not just out of `terms` — otherwise a
			// message that states them explicitly (e.g. "máy lạnh Daikin
			// treo tường ...") leaves them sitting in `remaining`, and
			// leftoverSearchTerm's multi-word case would re-wrap them as
			// one giant literal adjacent phrase ("máy lạnh Daikin treo
			// tường") on top of the already-correct terms above — which
			// then requires all 4 words adjacent in that exact order,
			// matching virtually no real product name. Verified as the
			// exact cause of a real 0-result case ("máy lạnh Daikin treo
			// tường 2HP giá từ 10 đến 20 triệu dùng gas R32 rẻ nhất" — 4
			// matching products exist in the DB, chat search returned 0).
			remaining = removePhraseWords(remaining, canonical)
			if hasBrand {
				remaining = removePhraseWords(remaining, brand)
			}
			if hasSub {
				remaining = removePhraseWords(remaining, sub)
			}
			// Same reasoning, same bug class: a named building type (see
			// buildingTypes) already does its job entirely through
			// CategorySlugs/heat-load-factor (applyCategorySlugs/
			// applyCapacity) — its keyword itself ("biệt thự", "hội
			// trường", ...) left sitting in `remaining` would otherwise
			// get swept into the leftover phrase too. Verified as a real
			// 0-of-19-missed case: "biệt thự nên lắp máy lạnh Daikin nào"
			// (no room size mentioned) returned only the 2 of 21 giấu-
			// trần Daikin products whose marketing copy happens to
			// literally say "biệt thự", instead of all 21 valid options.
			//
			// Strips every word used across the matched building type's
			// keywords (word-set, not phrase-sequence) — a message can
			// contain two overlapping keyword phrases for the same type
			// ("nhà xưởng sản xuất" contains both IND_FACTORY's "nhà
			// xưởng" and "xưởng sản xuất", sharing the word "xưởng").
			// Stripping "nhà xưởng" as a phrase first consumes "xưởng",
			// which then makes "xưởng sản xuất" impossible to find as a
			// contiguous run anymore — leaving "sản xuất" behind, wrapped
			// into its own mandatory phrase, and cutting 20m2's 11 correct
			// results down to 1. Word-set removal has no such ordering
			// dependency: every word belonging to this building type's
			// vocabulary is dropped regardless of which specific keyword
			// phrase it came from.
			// Strips *every* matching building type's word set, not just
			// the winning (highest heat_load_factor) one detectBuildingType
			// returns — a message naming two building types ("biệt thự nhà
			// phố 20m2") only has one drive the actual category/heat-load
			// choice, but both keywords still need to disappear from
			// `remaining`, or the losing one leaks into the leftover phrase
			// the exact same way.
			for _, bt := range buildingTypes {
				matched := false
				for _, kw := range bt.keywords {
					if strings.Contains(normalizeVietnamese(remaining), kw) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
				words := strings.Fields(remaining)
				kept := words[:0]
				for _, w := range words {
					if _, isBuildingWord := buildingTypeWordSets[bt.code][normalizeVietnamese(w)]; !isBuildingWord {
						kept = append(kept, w)
					}
				}
				remaining = strings.Join(kept, " ")
			}
			if extra := leftoverSearchTerm(stripStopwords(remaining)); extra != "" {
				terms = append(terms, extra)
			}
			filter.Search = quoteKnownPhrases(strings.Join(terms, " "))
			subName := ""
			if hasSub {
				subName = sub
			}
			explanation := buildExplanation(canonical, buildingTypeName, subName, capacity, brand, gas, filter.MinPrice, filter.MaxPrice, filter.SortBy)
			signals := chatQuerySignals{
				label:            label,
				canonical:        canonical,
				hasSub:           hasSub,
				sub:              sub,
				capacity:         capacity,
				gas:              gas,
				buildingTypeName: buildingTypeName,
				minPrice:         filter.MinPrice,
				maxPrice:         filter.MaxPrice,
				sortBy:           filter.SortBy,
			}
			return filter, narrowSearch, explanation, signals
		}
	}

	// Fallback path: no confident classifier prediction (unavailable
	// locally, or the message didn't clear the confidence bar) — same
	// regex/synonym table + stopword stripping as before the classifier
	// existed.
	text, capacity, gas := applyStructuredSignals(&filter, message)

	// Rewrite everyday phrasing to the vocabulary that actually appears in
	// product names, so search_vector (built from product name — see
	// ProductFilter.Search's doc comment) picks it up. Replaces in place
	// rather than appending: websearch_to_tsquery ANDs bare words together,
	// so leaving both "điều hòa" and "máy lạnh" in the query would demand
	// a name containing both, matching nothing.
	for _, syn := range categorySynonyms {
		if syn.pattern.MatchString(text) {
			text = syn.pattern.ReplaceAllString(text, syn.canonical)
			break
		}
	}

	filter.Search = quoteKnownPhrases(stripStopwords(text))
	// narrowSearch == filter.Search (not "") here on purpose: the fallback
	// path has no CategorySlugs guarantee (that's confident-classifier-
	// path only — see applyCategorySlugs), so relaxing an empty-result
	// fallback query down to an empty Search would mean "return every
	// published product with no filtering at all", not a sensible
	// narrower retry. Returning the same value as filter.Search makes
	// ChatSearchProducts' `narrowSearch != filter.Search` check correctly
	// skip the relaxation retry for this path. Bug caught in testing: an
	// off-topic fallback query ("giá vàng hôm nay") that legitimately
	// matched 0 products got "relaxed" all the way to 179 (every
	// published product) before this fix.
	//
	// No canonical category label to report here (that's the confident-
	// classifier path's job) — brand/building-type are still worth
	// detecting for the explanation even though this path doesn't use
	// them to build the filter itself.
	brand, _ := detectBrand(message)
	buildingTypeName := ""
	if bt, ok := detectBuildingType(message); ok {
		buildingTypeName = bt.name
	}
	explanation := buildExplanation("", buildingTypeName, "", capacity, brand, gas, filter.MinPrice, filter.MaxPrice, filter.SortBy)
	// label/canonical left "" here on purpose — buildSuggestions treats
	// that as "no confident category to dig deeper into" and falls back to
	// generic starting points instead (see its doc comment).
	signals := chatQuerySignals{
		capacity:         capacity,
		gas:              gas,
		buildingTypeName: buildingTypeName,
		minPrice:         filter.MinPrice,
		maxPrice:         filter.MaxPrice,
		sortBy:           filter.SortBy,
	}
	return filter, filter.Search, explanation, signals
}

// multiWordSearchPhrases are every canonical multi-word term chat search
// can place into a Search string: category names (categoryLabelCanonical/
// categorySynonyms) and AC sub-category form factors (subCategoryTerms).
// quoteKnownPhrases wraps their occurrences in double quotes before
// filter.Search reaches websearch_to_tsquery, which treats a quoted span as
// a phrase (adjacent lexemes, `'loc' <-> 'nuoc'`) instead of a bag of
// independently-matchable words.
//
// This matters because search_vector now indexes full marketing
// description text, not just the product name (migration 000018): a
// long, free-form description is much more likely to contain two common
// words *somewhere*, non-adjacent, than a short product name is. Verified
// in testing — "lọc nước" unquoted matched an unrelated Daikin AC product
// because its description separately mentioned "lọc không khí" (air
// filtering) and "chống đọng nước" (anti-condensation) — two unrelated
// marketing bullet points that happen to contain "lọc" and "nước". Quoted
// phrase search requires them adjacent, which only genuine "lọc nước"
// wording satisfies.
var multiWordSearchPhrases = []string{
	"máy lạnh", "lọc không khí", "lọc nước", "nhà thông minh",
	"âm trần", "giấu trần", "treo tường", "tủ đứng", "áp trần",
}

// multiWordSearchPhraseWords is multiWordSearchPhrases pre-split into
// lowercase word tokens, for the token-sequence matching quoteKnownPhrases
// does (see its doc comment for why this can't just be a `\b`-delimited
// regex: Go's RE2 \b only treats ASCII [0-9A-Za-z_] as "word" characters,
// so a boundary right after an accented letter like "í" — e.g. the end of
// "khí" — silently fails to match, missing exactly the Vietnamese phrases
// this exists to catch).
var multiWordSearchPhraseWords = buildMultiWordSearchPhraseWords()

func buildMultiWordSearchPhraseWords() [][]string {
	out := make([][]string, len(multiWordSearchPhrases))
	for i, phrase := range multiWordSearchPhrases {
		words := strings.Fields(phrase)
		for j, w := range words {
			words[j] = strings.ToLower(w)
		}
		out[i] = words
	}
	return out
}

// quoteKnownPhrases walks text word-by-word (not via regex — see
// multiWordSearchPhraseWords' doc comment) and wraps any run of words
// matching a known multi-word phrase in double quotes.
func quoteKnownPhrases(text string) string {
	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for i := 0; i < len(words); {
		matched := false
		for _, phraseWords := range multiWordSearchPhraseWords {
			n := len(phraseWords)
			if i+n > len(words) {
				continue
			}
			ok := true
			for j := 0; j < n; j++ {
				if strings.ToLower(words[i+j]) != phraseWords[j] {
					ok = false
					break
				}
			}
			if ok {
				out = append(out, `"`+strings.Join(words[i:i+n], " ")+`"`)
				i += n
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, words[i])
			i++
		}
	}
	return strings.Join(out, " ")
}

// removePhraseWords removes every occurrence of phrase's words as a
// contiguous, case-insensitive run from text — same word-token matching
// technique as quoteKnownPhrases, but deleting the run instead of quoting
// it. Used to strip a canonical/brand/sub-category term (already captured
// into Search's `terms` elsewhere) back out of the leftover text before
// that leftover becomes an "extra" feature-word phrase, so it doesn't get
// counted twice (see parseChatQuery's confident-classifier path).
func removePhraseWords(text, phrase string) string {
	phraseWords := strings.Fields(strings.ToLower(phrase))
	if len(phraseWords) == 0 {
		return text
	}
	words := strings.Fields(text)
	out := make([]string, 0, len(words))
	for i := 0; i < len(words); {
		n := len(phraseWords)
		if i+n <= len(words) {
			match := true
			for j, pw := range phraseWords {
				if strings.ToLower(words[i+j]) != pw {
					match = false
					break
				}
			}
			if match {
				i += n
				continue
			}
		}
		out = append(out, words[i])
		i++
	}
	return strings.Join(out, " ")
}

// applyStructuredSignals extracts every deterministic-rule signal from
// text into filter (price, AC capacity, gas type) and returns the text
// with those matched spans blanked out — shared by both the classifier
// and fallback paths since these are plain rules either way, independent
// of how the category gets decided.
// applyStructuredSignals also returns capacityInfo and the detected gas
// type ("" if none) — buildExplanation's raw material, so the caller can
// assemble a client-facing explanation of why these particular filters
// were chosen (see parseChatQuery).
func applyStructuredSignals(filter *domain.ProductFilter, text string) (string, capacityInfo, string) {
	text = applyPrice(filter, text)
	text, capacity := applyCapacity(filter, text)
	text, gas := applyGasType(filter, text)
	text = applySortDirective(filter, text)
	return text, capacity, gas
}

func applyPrice(filter *domain.ProductFilter, text string) string {
	switch {
	case rangePricePattern.MatchString(text):
		m := rangePricePattern.FindStringSubmatch(text)
		if v, ok := parsePriceVND(m[1]); ok {
			filter.MinPrice = &v
		}
		if v, ok := parsePriceVND(m[2]); ok {
			filter.MaxPrice = &v
		}
		text = rangePricePattern.ReplaceAllString(text, " ")
	case underPricePattern.MatchString(text):
		m := underPricePattern.FindStringSubmatch(text)
		if v, ok := parsePriceVND(m[1]); ok {
			filter.MaxPrice = &v
		}
		text = underPricePattern.ReplaceAllString(text, " ")
	case overPricePattern.MatchString(text):
		m := overPricePattern.FindStringSubmatch(text)
		if v, ok := parsePriceVND(m[1]); ok {
			filter.MinPrice = &v
		}
		text = overPricePattern.ReplaceAllString(text, " ")
	case barePricePattern.MatchString(text):
		// A lone number-of-millions with no "dưới"/"trên"/range wording is
		// almost always a shopper stating a budget ceiling.
		m := barePricePattern.FindStringSubmatch(text)
		if v, ok := parsePriceVND(m[1]); ok {
			filter.MaxPrice = &v
		}
		text = barePricePattern.ReplaceAllString(text, " ")
	}
	return text
}

// applyCapacity resolves an AC capacity tier from whichever form the
// shopper used — a directly-stated HP/ngựa figure (highest priority: if
// they said it, trust it over any derived guess), room dimensions
// ("4x5m"), volume ("60m3" — converted via the same BTU≈area×600 rule as
// roomAreaToHPTier, expressed as an equivalent area assuming a standard
// ~3m ceiling: BTU≈volume×200 = (volume/3)×600), or plain area ("20m2") —
// and applies it as a phan_khuc_hp attribute token, using the existing
// structured attribute facet system instead of hoping the wording happens
// to match product-name text. Each matched span is blanked out afterwards
// for the same AND-everything reason as the price patterns.
//
// When the tier comes from a derived measurement (not a directly-stated
// HP), a heat-load cue in the message (direct sun, a crowded venue with
// doors opening constantly, an adjoining kitchen, 24/7 unattended
// operation, ...) inflates the area by heatLoadAreaMultiplier *before* the
// tier lookup — this is a coarse heuristic, not real thermal-load
// reasoning (that would need actual inference, out of scope without an
// LLM), but it beats silently ignoring the context entirely.
//
// Applying the adjustment to the area pre-lookup (rather than bumping the
// resulting discrete tier by one step afterwards, the original design)
// avoids over-provisioning at a bracket's low edge: a room right at 15m²
// with a heat-load cue previously jumped a full tier to 2 HP, when
// 15m²×1.25 = 18.75m² is still comfortably inside the 15-20m² (1.5 HP)
// bracket — a real published sizing table (see roomAreaToHPTier's doc
// comment) already bakes in a safety margin at its bracket edges, so
// stacking a second full-tier jump on top double-counts that margin.
// Verified this doesn't under-provision the case at the other end either:
// a 40m² room with a heat-load cue still resolves to 3.5 HP either way
// (40×1.25=50m², same bracket the old discrete-bump path landed in).

// capacityKind labels which rule produced capacityInfo.tier — buildExplanation
// uses it to render a *different* sentence shape per signal (a direct HP
// figure needs no math shown; a dimension needs the multiplication spelled
// out; a volume needs the m³->m² conversion spelled out; ...), rather than
// one generic "công suất đề xuất X" line that hides how the number was
// actually reached.
type capacityKind int

const (
	capacityKindNone capacityKind = iota
	capacityKindDirect
	capacityKindArea
	capacityKindDimension
	capacityKindVolume
)

// capacityInfo carries the pieces buildExplanation needs to describe how
// the phan_khuc_hp tier was chosen — separate from the AttributeTokens
// string encoding so the explanation builder doesn't have to parse that
// back out. rawArea/length/width/volume are the *pre* heat-load-adjustment
// values (what the shopper actually stated), so the explanation can show
// both "you said Xm²" and "adjusted to Y for Z" as distinct steps.
type capacityInfo struct {
	kind             capacityKind
	tier             string
	rawArea          float64 // set for kindArea/kindDimension (already length*width)/kindVolume (already volume/3 equivalent)
	length, width    float64 // set for kindDimension only
	volume           float64 // set for kindVolume only
	buildingTypeName string // "" if no named building type drove the heat-load adjustment
	heatLoadPercent  int    // 0 if no heat-load adjustment applied at all
}

func applyCapacity(filter *domain.ProductFilter, text string) (string, capacityInfo) {
	text = normalizeDecimeterNotation(text)

	var info capacityInfo
	var derivedArea float64

	switch {
	case hpMentionPattern.MatchString(text):
		m := hpMentionPattern.FindStringSubmatch(text)
		if hp, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64); err == nil {
			info.kind = capacityKindDirect
			info.tier = formatHPTier(hp)
		}
		text = hpMentionPattern.ReplaceAllString(text, " ")
	case dimensionPattern.MatchString(text):
		m := dimensionPattern.FindStringSubmatch(text)
		length, errL := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		width, errW := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
		if errL == nil && errW == nil {
			info.kind = capacityKindDimension
			info.length, info.width = length, width
			info.rawArea = length * width
			derivedArea = info.rawArea
		}
		text = dimensionPattern.ReplaceAllString(text, " ")
	case labeledLengthWidthPattern.MatchString(text) || labeledWidthLengthPattern.MatchString(text):
		var m []string
		if labeledLengthWidthPattern.MatchString(text) {
			m = labeledLengthWidthPattern.FindStringSubmatch(text)
		} else {
			m = labeledWidthLengthPattern.FindStringSubmatch(text)
		}
		a, errA := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		b, errB := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
		if errA == nil && errB == nil {
			info.kind = capacityKindDimension
			info.length, info.width = a, b
			info.rawArea = a * b
			derivedArea = info.rawArea
		}
		text = labeledLengthWidthPattern.ReplaceAllString(text, " ")
		text = labeledWidthLengthPattern.ReplaceAllString(text, " ")
	case volumePattern.MatchString(text):
		m := volumePattern.FindStringSubmatch(text)
		if volume, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64); err == nil {
			info.kind = capacityKindVolume
			info.volume = volume
			info.rawArea = volume / 3
			derivedArea = info.rawArea
		}
		text = volumePattern.ReplaceAllString(text, " ")
	case roomAreaPattern.MatchString(text):
		m := roomAreaPattern.FindStringSubmatch(text)
		if area, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64); err == nil {
			info.kind = capacityKindArea
			info.rawArea = area
			derivedArea = area
		}
		text = roomAreaPattern.ReplaceAllString(text, " ")
	}

	if info.kind != capacityKindNone && info.kind != capacityKindDirect {
		// A named building type (see buildingTypes) carries its own
		// known heat-load multiplier, sharper than the flat
		// heatLoadAreaMultiplier catch-all — e.g. a restaurant's hot
		// kitchen (1.4x) genuinely needs more margin than an office
		// (1.1x), even though both would just trigger the same generic
		// bump under hasHeatLoadCue alone. Checked first so a named
		// building type's own factor wins when both would otherwise
		// apply.
		switch bt, ok := detectBuildingType(text); {
		case ok:
			derivedArea *= bt.heatLoadFactor
			info.buildingTypeName = bt.name
			info.heatLoadPercent = percentOver100(bt.heatLoadFactor)
		case hasHeatLoadCue(text):
			derivedArea *= heatLoadAreaMultiplier
			info.heatLoadPercent = percentOver100(heatLoadAreaMultiplier)
		}
		if t, ok := roomAreaToHPTier(derivedArea); ok {
			info.tier = t
		}
	}

	if info.tier == "" {
		return text, capacityInfo{}
	}
	filter.AttributeTokens = append(filter.AttributeTokens, "phan_khuc_hp:"+info.tier)
	return text, info
}

// percentOver100 converts a multiplier like 1.25 to 25 (the "+X%" a
// shopper actually reads) — used by buildExplanation.
func percentOver100(factor float64) int {
	return int(factor*100+0.5) - 100
}

// applyGasType maps a mentioned refrigerant ("gas R32", or just "R32") to
// the loai_gas_lanh attribute token — a real select-type attribute in the
// DB (options: R32/R410A/R22/R290), not a guess.
func applyGasType(filter *domain.ProductFilter, text string) (string, string) {
	gas, ok := detectGasType(text)
	if !ok {
		return text, ""
	}
	filter.AttributeTokens = append(filter.AttributeTokens, "loai_gas_lanh:"+gas)
	text = gasTypePattern.ReplaceAllString(text, " ")
	return text, gas
}

// categoryLabelSlugs maps a classifier label to every category slug filed
// under it (see the `categories` table — group "Máy lạnh" has 5 form-
// factor categories, the others map 1:1 or many:1). Used as a hard
// CategorySlugs filter (domain.ProductFilter.CategorySlugs): a product can
// only match if it's genuinely filed under one of these categories,
// independent of anything its Search text/description happens to say —
// stronger than Search-text matching alone, which even with phrase
// quoting (quoteKnownPhrases) is still a text heuristic, not a guarantee.
// Update when a category is added/removed/renamed (mirrors
// categoryLabelCanonical's own maintenance note).
var categoryLabelSlugs = map[string][]string{
	"may_lanh": {
		"may-lanh-treo-tuong", "may-lanh-am-tran-da-huong-thoi",
		"may-lanh-giau-tran-noi-ong-gio", "may-lanh-tu-dung", "may-lanh-ap-tran",
	},
	"loc_khong_khi": {"may-cap-khi-tuoi-loc-khong-khi"},
	"loc_nuoc":      {"may-loc-nuoc-ro-3-in-1"},
	"nha_thong_minh": {
		"bang-dieu-khien", "cong-tac-thong-minh", "cam-bien-thong-minh", "remote-cam-tay",
	},
}

// subCategorySlugs maps an AC sub-category's canonical term (subCategoryTerms)
// to its specific category slug — an explicit form-factor mention narrows
// CategorySlugs down to exactly that one category instead of the whole
// may_lanh group.
var subCategorySlugs = map[string]string{
	"âm trần":    "may-lanh-am-tran-da-huong-thoi",
	"giấu trần":  "may-lanh-giau-tran-noi-ong-gio",
	"treo tường": "may-lanh-treo-tuong",
	"tủ đứng":    "may-lanh-tu-dung",
	"áp trần":    "may-lanh-ap-tran",
}

// maxTreoTuongTier is the largest phan_khuc_hp tier elc's wall-mount
// ("treo tường") category actually carries (verified against real product
// data: treo tường tops out at 3 HP there; larger capacities are only
// cassette/floor-standing/duct units). Gates applyCategorySlugs' default
// below so a large-room query doesn't get squeezed into a category that
// can't actually serve it.
const maxTreoTuongTier = "3 HP"

// accessoryIntentPattern matches phrasing for "I already own the main
// unit, I just need parts" — see applyCategorySlugs.
var accessoryIntentPattern = regexp.MustCompile(`(?i)phụ\s*kiện|phu\s*kien|vật\s*tư|vat\s*tu|đường\s*ống|duong\s*ong|thi\s*công|thi\s*cong`)

// applyCategorySlugs sets filter.CategorySlugs. For every category label
// except "may_lanh" this is a direct lookup (categoryLabelSlugs). For
// "may_lanh" specifically:
//   - an explicit form factor ("âm trần", "tủ đứng", ...) narrows straight
//     to that one category (subCategorySlugs).
//   - otherwise, when the requested/derived capacity is within what
//     "treo tường" (wall-mount split) carries — or no capacity was stated
//     at all — default to treo tường: it's the single largest AC category
//     in elc's catalog (59 of ~135 AC products) and the type Vietnamese
//     retail means by default when a shopper just says "máy lạnh" with no
//     further detail. Someone who wants a cassette/floor-standing/duct
//     unit either names it explicitly (handled above) or is asking for a
//     capacity treo tường doesn't carry, which falls through to the full
//     may_lanh group below instead of silently returning zero results.
func applyCategorySlugs(filter *domain.ProductFilter, label, sub, message string, hasSub bool) {
	// "I already have the main unit, I just need accessories/installation
	// parts" — e.g. "phụ kiện đường ống thi công âm trần" for a fresh-air
	// system already owned. Without this, a loc_khong_khi query always
	// resolved to the main-unit category and never surfaced the dedicated
	// accessories category (real products that exist in the catalog —
	// ducting, vents, distribution boxes — were unreachable via chat
	// search even though they're genuinely there).
	if label == "loc_khong_khi" && accessoryIntentPattern.MatchString(message) {
		filter.CategorySlugs = []string{"phu-kien-dong-bo-cua-he-thong-cap-gio-tuoi"}
		return
	}
	if label != "may_lanh" {
		if slugs, ok := categoryLabelSlugs[label]; ok {
			filter.CategorySlugs = slugs
		}
		return
	}
	if hasSub {
		if slug, ok := subCategorySlugs[sub]; ok {
			filter.CategorySlugs = []string{slug}
			return
		}
	}
	// A named building type (see buildingTypes) with a real elc
	// equivalent overrides the generic treo-tường default — e.g. a
	// restaurant/cafe genuinely fits "âm trần" (cassette) far better than
	// wall-mount, and a villa/hotel fits "giấu trần" (concealed duct)
	// for its aesthetic/quiet requirements. Checked after an explicit
	// user-stated form factor (hasSub above), which always wins outright.
	// mappedCategorySlug is "" for building types elc has no honest
	// category for (hospital, cleanroom, data center, factory, large
	// event hall/church) — those fall through to the normal default
	// below rather than forcing an ill-fitting match.
	if bt, ok := detectBuildingType(message); ok && bt.mappedCategorySlug != "" {
		filter.CategorySlugs = []string{bt.mappedCategorySlug}
		return
	}
	if withinTreoTuongRange(filter.AttributeTokens) {
		filter.CategorySlugs = []string{"may-lanh-treo-tuong"}
		return
	}
	filter.CategorySlugs = categoryLabelSlugs["may_lanh"]
}

// withinTreoTuongRange reports whether filter's derived phan_khuc_hp tier
// (if any) is small enough for the treo-tường default (see
// applyCategorySlugs) — true when there's no capacity signal at all, since
// that means no room-size info to contradict the default.
func withinTreoTuongRange(attrTokens []string) bool {
	for _, tok := range attrTokens {
		code, value, ok := strings.Cut(tok, ":")
		if ok && code == "phan_khuc_hp" {
			return hpTierIndex(value) <= hpTierIndex(maxTreoTuongTier)
		}
	}
	return true
}

// hpTierIndex returns tier's position in hpTierOrder (ascending capacity),
// or -1 if unrecognized.
func hpTierIndex(tier string) int {
	for i, t := range hpTierOrder {
		if t == tier {
			return i
		}
	}
	return -1
}

// applySortDirective detects a superlative ranking request — "rẻ nhất"
// (cheapest), "đắt nhất"/"cao cấp nhất" (priciest/flagship), "mới nhất"
// (newest) — and sets filter.SortBy accordingly, blanking the matched span
// same as every other applyStructuredSignals step: left in, "đắt"/"rẻ"/
// "mới" would survive into Search as ordinary leftover content words and
// AND-pollute the query (a product's description rarely contains the bare
// word "đắt"). A small, closed set of fixed Vietnamese phrasings, not
// open-ended language understanding, so plain rules cover it same as
// everything else in this file.
func applySortDirective(filter *domain.ProductFilter, text string) string {
	switch {
	case cheapestPattern.MatchString(text):
		filter.SortBy = domain.SortByPriceAsc
		text = cheapestPattern.ReplaceAllString(text, " ")
	case priciestPattern.MatchString(text):
		filter.SortBy = domain.SortByPriceDesc
		text = priciestPattern.ReplaceAllString(text, " ")
	case newestPattern.MatchString(text):
		filter.SortBy = domain.SortByNewest
		text = newestPattern.ReplaceAllString(text, " ")
	}
	return text
}

var (
	cheapestPattern = regexp.MustCompile(`(?i)(?:rẻ|re|thấp|thap)\s*(?:nhất|nhat)`)
	priciestPattern = regexp.MustCompile(`(?i)(?:đắt|dat|cao\s*cấp|cao\s*cap|xịn|xin)\s*(?:nhất|nhat)`)
	newestPattern   = regexp.MustCompile(`(?i)(?:mới|moi)\s*(?:nhất|nhat|ra\s*mắt|ra\s*mat)`)
)

// roomAreaPattern matches "20m2" / "20 m2" / "20m²" / "20 mét vuông" / "20
// m vuông", optionally preceded by "phòng". The room-size number is almost
// never ambiguous with a price mention (those require "tr"/"triệu"), so no
// price/area pattern ordering dependency here.
var roomAreaPattern = regexp.MustCompile(`(?i)(?:phòng\s*)?(\d+(?:[.,]\d+)?)\s*(?:m2|m²|m\s*vuông|mét\s*vuông)\b`)

// decimeterNotationPattern matches the casual Vietnamese "meters-decimeter"
// way of writing a length, most familiar from stating someone's height
// ("1m75" = 1.75m) — e.g. "3m8" meaning 3.8m. Digits 2 and 3 are excluded
// from the second group on purpose: "20m2"/"60m3" are already meaningful,
// far more common units (square/cubic meters — see roomAreaPattern/
// volumePattern), so "Xm2"/"Xm3" must keep meaning "X square/cubic
// meters", not get reinterpreted as "X.2 meters"/"X.3 meters". This can't
// disambiguate a genuine "3m2" meant as "3.2 mét" from "3 mét vuông" — in
// practice the area/volume reading is overwhelmingly the intended one for
// digits 2/3, so that ambiguity is resolved in area/volume's favor.
var decimeterNotationPattern = regexp.MustCompile(`(\d+)m([014-9])\b`)

// normalizeDecimeterNotation rewrites decimeterNotationPattern matches
// ("3m8") into a bare decimal number ("3.8", no unit suffix) before
// dimension/area/volume patterns run, so a phrase like "phòng 3m8 x 6m"
// (originally misread as dimensions "8 x 6" = 48m², since "3m8 x 6m"
// contains "8 x 6m" as a substring once the leading "3m" is skipped over)
// resolves to the intended 3.8 x 6 = 22.8m² instead. Deliberately no
// trailing "m" on the replacement: dimensionPattern only expects a unit
// suffix on the *second* number ("4x5m", not "4mx5m") — keeping one here
// would make "3.8m x 6m" fail to match at all (verified: caused
// dimensionPattern to silently not match after the very first version of
// this fix).
func normalizeDecimeterNotation(text string) string {
	// ${2} (not $2) is required even without the trailing "m" once there's
	// any following text that could look like a group name — Go's regexp
	// ReplaceAllString parses "$2m" as a reference to a group literally
	// NAMED "2m" (which doesn't exist, so it silently expands to ""), not
	// "group 2" followed by a literal "m". Verified the hard way: "$1.$2m"
	// on "3m8" produced "3." — the "8" and "m" both vanished. ${2} avoids
	// the ambiguity regardless of what (if anything) follows in the
	// replacement string.
	return decimeterNotationPattern.ReplaceAllString(text, "$1.${2}")
}

// hpMentionPattern matches a directly-stated capacity: "1.5 HP", "1.5hp",
// "2 ngựa", "2ngua". Checked before the area/dimension/volume patterns in
// applyCapacity — an explicit figure the shopper stated outranks anything
// derived from a room measurement.
var hpMentionPattern = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(?:hp|ngựa|ngua)\b`)

// dimensionPattern matches room dimensions given as length x width in
// meters: "4x5m", "4 x 5 m", "4x5 mét", and — the unit optional on the
// *first* number too — "4m x 5m" (a real, arguably more common way to
// state it; the bare-first-number form alone missed this entirely: "2m x
// 4m" has an "m" sitting right after "2", so requiring `\s*x` immediately
// after the first digit group never matched at all, silently dropping the
// capacity signal). Multiplied into an area and fed through the same
// roomAreaToHPTier table as a plain "Xm2" mention.
var dimensionPattern = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(?:m|mét)?\s*x\s*(\d+(?:[.,]\d+)?)\s*(?:m|mét)\b`)

// labeledLengthWidthPattern/labeledWidthLengthPattern match room dimensions
// given with explicit "dài"/"sâu"/"rộng"/"ngang" (length/depth/width)
// labels — with or without the "chiều" ("chiều dài"/"chiều rộng") prefix,
// since real messages use all of these ("chiều dài 2m chiều ngang 10m",
// the terser "dài 4m rộng 10m", and "rộng 3m sâu 8m") — instead of the
// compact "AxBm" form dimensionPattern handles. Real queries originally
// slipped through unmatched this way (no capacity signal extracted at all,
// so results spanned every HP tier with no room-size filter applied):
// neither dimensionPattern (needs a literal "x") nor roomAreaPattern
// (needs an "m2"/"mét vuông" suffix on a single number) matches any of
// this phrasing. Two patterns cover either stated-dimension order; which
// capture is length vs width doesn't matter since only their product (the
// area) is used. `[^\d]{0,15}` bridges the words between the two clauses
// without risking a match across an unrelated second number elsewhere in
// a longer message.
var (
	labeledLengthWidthPattern = regexp.MustCompile(`(?i)(?:chiều\s*)?(?:dài|sâu)\s*(\d+(?:[.,]\d+)?)\s*m?\b[^\d]{0,15}(?:chiều\s*)?(?:ngang|rộng)\s*(\d+(?:[.,]\d+)?)\s*m?\b`)
	labeledWidthLengthPattern = regexp.MustCompile(`(?i)(?:chiều\s*)?(?:ngang|rộng)\s*(\d+(?:[.,]\d+)?)\s*m?\b[^\d]{0,15}(?:chiều\s*)?(?:dài|sâu)\s*(\d+(?:[.,]\d+)?)\s*m?\b`)
)

// volumePattern matches room volume: "60m3", "60 m3", "60m³", "60 khối".
var volumePattern = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(?:m3|m³|khối|khoi)\b`)

// hpTierOrder is every phan_khuc_hp option in ascending capacity order —
// used by hpTierIndex (see withinTreoTuongRange).
var hpTierOrder = []string{
	"1 HP", "1.5 HP", "2 HP", "2.5 HP", "3 HP",
	"3.5 HP", "4 HP", "4.5 HP", "5 HP", "5.5 HP",
}

// formatHPTier renders a directly-stated HP figure ("2", "2.5") to match
// the phan_khuc_hp attribute's exact option format ("2 HP", "2.5 HP").
func formatHPTier(hp float64) string {
	if hp == math.Trunc(hp) {
		return strconv.FormatFloat(hp, 'f', 0, 64) + " HP"
	}
	return strconv.FormatFloat(hp, 'f', 1, 64) + " HP"
}

// heatLoadAreaMultiplier inflates a derived room area by this factor when
// hasHeatLoadCue matches — see applyCapacity's doc comment for why this
// applies pre-tier-lookup rather than bumping the resulting tier by one
// step. 1.25 (a 25% BTU bump) is the midpoint of the +20%–+30% range
// commonly cited for direct-sun/west-facing rooms in Vietnamese AC sizing
// guidance.
const heatLoadAreaMultiplier = 1.25

// heatLoadKeywords are context cues (checked via normalizeVietnamese, so
// accents/case don't need listing separately) that mean a room needs more
// cooling capacity than its bare size suggests: direct sun exposure, poor
// insulation, a high-traffic venue with doors opening constantly, heat
// from an adjoining kitchen, or 24/7 unattended operation. Approximate by
// design — see applyCapacity's doc comment.
var heatLoadKeywords = []string{
	"nang chieu", "huong tay", "huong dong", "tuong mong", "tuong gach mong",
	"dong nguoi", "mo cua lien tuc", "quan net", "quan bida", "quan game",
	"nha kho", "phong server", "24/7", "chay 24", "hoat dong lien tuc",
	"bep nau", "nau nuong", "gym", "yoga", "phong tap",
	// Roof/direct-sun phrasing found via a real failing query ("mái tôn
	// hất nắng xuống cực nóng buổi trưa") — "mái tôn" (galvanized sheet
	// roofing, notorious in Vietnam for radiating heat straight through
	// into the room below) and "cực nóng"/"hất nắng"/"nắng hắt" weren't
	// covered by the existing "nắng chiếu"/"hướng tây" cues.
	"mai ton", "cuc nong", "hat nang", "nang hat", "nong buc",
}

func hasHeatLoadCue(text string) bool {
	normalized := normalizeVietnamese(text)
	for _, kw := range heatLoadKeywords {
		if strings.Contains(normalized, kw) {
			return true
		}
	}
	return false
}

// buildingType is one entry from an internal HVAC building taxonomy
// (villa/apartment/townhouse/cafe/restaurant/retail/office/hotel/event-
// hall/church/hospital/cleanroom/data-center/factory) — sharper than the
// generic heatLoadKeywords catch-all because each building type carries
// its own known heat-load multiplier instead of one flat guess.
//
// mappedCategorySlug is deliberately empty for building types the real
// elc catalog has no honest answer for. elc's actual inventory is 100%
// residential/light-commercial split AC across 5 form factors, up to 5.5
// HP (see categoryLabelSlugs) — it carries no VRV/VRF, chiller, AHU,
// precision/cleanroom cooling, or industrial package units, even though
// the source taxonomy assumes a catalog that does. Forcing a hospital,
// cleanroom, data center, factory, or large event-hall/church query onto
// "tủ đứng" or "âm trần" just because those are elc's largest-capacity
// categories would be a misleading recommendation — a single split unit
// isn't a substitute for central plant equipment — so those entries only
// sharpen the heat-load multiplier and leave category selection to fall
// through to the normal (may_lanh-wide or treo-tường-default) logic
// instead of claiming a fit that isn't there.
var buildingTypes = []struct {
	code               string
	name               string // Vietnamese display name, for the client-facing sizing note (see buildingTypeMatch.name)
	keywords           []string
	heatLoadFactor     float64
	mappedCategorySlug string
}{
	{"RES_VILLA", "biệt thự/villa", []string{"biet thu", "villa", "dinh thu", "penthouse", "duplex", "nha vuon cao cap"}, 1.2, "may-lanh-giau-tran-noi-ong-gio"},
	{"RES_APT_HIGH", "chung cư cao cấp", []string{"chung cu cao cap", "can ho cao cap", "condotel", "chung cu"}, 1.0, ""},
	{"RES_TOWN", "nhà phố/nhà ống", []string{"nha pho", "nha ong", "nha lech tang", "shophouse", "nha lien ke"}, 1.15, "may-lanh-treo-tuong"},
	{"COM_CAFE", "quán cafe/trà sữa", []string{"quan cafe", "ca phe", "quan tra sua", "tiem banh", "bistro", "quan nuoc", "coffee shop"}, 1.3, "may-lanh-am-tran-da-huong-thoi"},
	{"COM_REST", "nhà hàng/quán ăn", []string{"nha hang", "quan an", "quan nhau", "tiem lau", "buffet", "quan nuong"}, 1.4, "may-lanh-am-tran-da-huong-thoi"},
	{"COM_RETAIL", "showroom/cửa hàng", []string{"showroom", "shop thoi trang", "cua hang quan ao", "boutique", "sieu thi mini"}, 1.25, "may-lanh-am-tran-da-huong-thoi"},
	{"EVT_HALL", "hội trường/trung tâm sự kiện", []string{"trung tam tiec cuoi", "hoi truong", "nha thi dau", "trung tam hoi nghi", "ballroom", "san tiec"}, 1.5, ""},
	{"EVT_CHURCH", "nhà thờ/chùa", []string{"nha tho", "thanh that", "chua", "thien vien", "giang duong lon"}, 1.3, ""},
	{"OFF_BLOCK", "văn phòng", []string{"van phong", "co-working", "coworking", "tru so", "phong hop"}, 1.1, "may-lanh-am-tran-da-huong-thoi"},
	{"HOSP_HOTEL", "khách sạn/resort", []string{"khach san", "hotel", "resort", "homestay", "nha nghi", "khu nghi duong"}, 1.1, "may-lanh-giau-tran-noi-ong-gio"},
	{"MED_HOSP", "bệnh viện/phòng khám", []string{"benh vien", "phong kham", "nha khoa", "tham my vien", "phong mo", "phong cap cuu"}, 1.2, ""},
	{"MED_CLEAN", "phòng sạch/phòng lab", []string{"phong sach", "cleanroom", "phong lab", "phong thi nghiem", "nha may duoc"}, 1.4, ""},
	{"IND_SERVER", "phòng server/data center", []string{"phong server", "data center", "phong may chu", "phong mdf", "phong idf", "tram bts"}, 1.8, ""},
	{"IND_FACTORY", "nhà xưởng/kho hàng", []string{"nha xuong", "xuong san xuat", "kho hang", "bai xe trong nha", "xuong may"}, 1.5, ""},
}

type buildingTypeMatch struct {
	code               string
	name               string
	heatLoadFactor     float64
	mappedCategorySlug string
}

// detectBuildingType looks for a known project/venue type anywhere in
// text — same normalizeVietnamese-based matching as detectBrand/
// detectSubCategory, so accents/case don't need listing separately.
// A message can name more than one building type at once ("biệt thự nhà
// phố 20m2" mentions both RES_VILLA and RES_TOWN). detectBuildingType
// resolves that by picking the match with the highest heatLoadFactor, not
// whichever happened to match first — sizing for the more demanding
// characteristic is the physically correct call (a room that's both a
// villa and street-facing needs at least as much capacity as either alone
// implies), and it's also the only rule that doesn't silently depend on
// buildingTypes' declaration order or the words' order in the message.
// Verified this matters in practice: "quán cafe văn phòng 20m2" and "văn
// phòng quán cafe 20m2" must resolve the same way regardless of which
// phrase comes first in the sentence.
func detectBuildingType(text string) (buildingTypeMatch, bool) {
	normalized := normalizeVietnamese(text)
	var best buildingTypeMatch
	found := false
	for _, bt := range buildingTypes {
		for _, kw := range bt.keywords {
			if strings.Contains(normalized, kw) {
				if !found || bt.heatLoadFactor > best.heatLoadFactor {
					best = buildingTypeMatch{bt.code, bt.name, bt.heatLoadFactor, bt.mappedCategorySlug}
					found = true
				}
				break
			}
		}
	}
	return best, found
}

// buildingTypeWordSets is buildingTypes' keywords pre-split into
// per-code word sets, for parseChatQuery's leftover-text stripping — see
// its doc comment for why word-set membership (not phrase-sequence
// removal) is the correct technique when a building type has multiple
// overlapping keyword phrases.
var buildingTypeWordSets = buildBuildingTypeWordSets()

func buildBuildingTypeWordSets() map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{}, len(buildingTypes))
	for _, bt := range buildingTypes {
		set := make(map[string]struct{})
		for _, kw := range bt.keywords {
			for _, w := range strings.Fields(kw) {
				set[w] = struct{}{}
			}
		}
		out[bt.code] = set
	}
	return out
}

// gasTypes maps a mentioned refrigerant to the loai_gas_lanh attribute's
// exact option value (R32/R410A/R22/R290).
var gasTypes = []struct {
	pattern *regexp.Regexp
	value   string
}{
	{regexp.MustCompile(`(?i)\br32\b`), "R32"},
	{regexp.MustCompile(`(?i)\br410a\b`), "R410A"},
	{regexp.MustCompile(`(?i)\br22\b`), "R22"},
	{regexp.MustCompile(`(?i)\br290\b`), "R290"},
}

// gasTypePattern matches any of gasTypes' patterns, for blanking out the
// matched span once a value's been captured via detectGasType.
var gasTypePattern = regexp.MustCompile(`(?i)\b(?:gas\s*)?r(?:32|410a|22|290)\b`)

func detectGasType(text string) (string, bool) {
	for _, g := range gasTypes {
		if g.pattern.MatchString(text) {
			return g.value, true
		}
	}
	return "", false
}

// subCategoryTerms maps AC form-factor phrasing to the term that appears
// in that sub-category's product names (see the `categories` table under
// the "Máy lạnh" group: treo tường/âm trần/giấu trần/tủ đứng/áp trần).
// Unlike categorySynonyms, this is *appended* to the category canonical
// term rather than replacing it — "âm trần" is real product-name
// vocabulary, not filler, so it narrows the AND-match safely (see
// detectBrand's doc comment for the same reasoning applied to brands).
var subCategoryTerms = []struct {
	canonical string
	aliases   []string
}{
	{"âm trần", []string{"âm trần cassette", "âm trần", "cassette"}},
	{"giấu trần", []string{"giấu trần nối ống gió", "giấu trần", "slim duct", "noi ong gio"}},
	{"treo tường", []string{"treo tường"}},
	{"tủ đứng", []string{"tủ đứng"}},
	{"áp trần", []string{"áp trần"}},
}

type subCategoryRule struct {
	pattern   *regexp.Regexp
	canonical string
}

var subCategoryPatterns = buildSubCategoryPatterns()

func buildSubCategoryPatterns() []subCategoryRule {
	var rules []subCategoryRule
	for _, s := range subCategoryTerms {
		for _, alias := range s.aliases {
			rules = append(rules, subCategoryRule{
				pattern:   regexp.MustCompile(regexp.QuoteMeta(normalizeVietnamese(alias))),
				canonical: s.canonical,
			})
		}
	}
	return rules
}

func detectSubCategory(text string) (string, bool) {
	normalized := normalizeVietnamese(text)
	for _, s := range subCategoryPatterns {
		if s.pattern.MatchString(normalized) {
			return s.canonical, true
		}
	}
	return "", false
}

// roomAreaToHPTier maps a room area (m²) to the smallest AC capacity tier
// that comfortably cools it, matching the phan_khuc_hp attribute's exact
// option strings ("1 HP", "1.5 HP", ...). Breakpoints under 40m² are the
// widely-published Điện Máy Xanh sizing table (the de facto standard
// referenced across Vietnamese retailers); above that they're extrapolated
// from the same BTU ≈ area × 600 rule of thumb the source articles give,
// since published tables get sparse/inconsistent past ~40m².
//
// Boundaries are exclusive on the low end to match how the source states
// them ("1 HP: under 15m²", "1.5 HP: 15-20m²") — so a room of exactly 15m²
// or 20m² rounds up into the next tier, not down, which also matches the
// same sources' general "when in doubt, size up ~0.5-1 HP" advice.
func roomAreaToHPTier(area float64) (string, bool) {
	switch {
	case area <= 0:
		return "", false
	case area < 15:
		return "1 HP", true
	case area < 20:
		return "1.5 HP", true
	case area < 30:
		return "2 HP", true
	case area < 40:
		return "2.5 HP", true
	case area < 46:
		return "3 HP", true
	case area < 53:
		return "3.5 HP", true
	case area < 60:
		return "4 HP", true
	case area < 68:
		return "4.5 HP", true
	case area < 75:
		return "5 HP", true
	default:
		return "5.5 HP", true
	}
}

func parsePriceVND(s string) (int64, bool) {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0, false
	}
	return int64(f * 1_000_000), true
}

// buildExplanation assembles a plain-language Vietnamese sentence
// describing why these particular filters were chosen — template-based,
// not generative: every clause below is inserted from data the rules
// above already computed (category, building type, capacity tier, brand,
// gas type, price range, sort order), not written by a language model.
// Empty inputs are simply omitted from the sentence, so a bare "máy lạnh"
// query still gets a short, honest "Gợi ý máy lạnh." rather than a
// padded-out non-answer.
func buildExplanation(category, buildingTypeName, subName string, capacity capacityInfo, brand, gas string, minPrice, maxPrice *int64, sortBy string) string {
	head := "Gợi ý sản phẩm phù hợp"
	if category != "" {
		head = "Gợi ý " + category
	}
	if buildingTypeName != "" {
		head += " cho " + buildingTypeName
	}
	if subName != "" {
		head += " (" + subName + ")"
	}
	parts := []string{head}

	if s := capacityClause(capacity); s != "" {
		parts = append(parts, s)
	}
	if brand != "" {
		parts = append(parts, "thương hiệu "+brand)
	}
	if gas != "" {
		parts = append(parts, "dùng gas "+gas)
	}
	switch {
	case minPrice != nil && maxPrice != nil:
		parts = append(parts, fmt.Sprintf("giá từ %s đến %s", formatVNDTrieu(*minPrice), formatVNDTrieu(*maxPrice)))
	case maxPrice != nil:
		parts = append(parts, fmt.Sprintf("giá dưới %s", formatVNDTrieu(*maxPrice)))
	case minPrice != nil:
		parts = append(parts, fmt.Sprintf("giá trên %s", formatVNDTrieu(*minPrice)))
	}
	switch sortBy {
	case domain.SortByPriceAsc:
		parts = append(parts, "sắp xếp theo giá thấp đến cao")
	case domain.SortByPriceDesc:
		parts = append(parts, "sắp xếp theo giá cao đến thấp")
	case domain.SortByNewest:
		parts = append(parts, "sắp xếp theo sản phẩm mới nhất")
	}
	return strings.Join(parts, ", ") + "."
}

// chatSuggestionLimit caps how many follow-up chips a turn surfaces — same
// "a few good options, not a wall of choices" reasoning as ChatSearchLimit.
const chatSuggestionLimit = 3

// buildSuggestions proposes follow-up queries a shopper can tap to dig
// deeper from wherever this turn's answer left off — one candidate per
// still-open decision (form factor, room size/capacity, budget, gas type,
// ranking), each phrased as a complete, self-contained query (never a bare
// question) by appending one new qualifier onto everything this turn's
// message already established. That "context so far" phrase, not a fixed
// wording, is what makes each round's chips read as a genuine deepening of
// *this* conversation instead of the same generic starter prompts
// repeating — e.g. after "máy lạnh âm trần cho quán cafe", the missing-
// capacity candidate becomes "máy lạnh âm trần cho quán cafe cho phòng
// 20m2", not a generic "phòng bao nhiêu m²?".
//
// Grounded in elc's real catalog only — residential/light-commercial split
// AC across the 5 known form factors, up to 5.5 HP (categoryLabelSlugs/
// maxTreoTuongTier) — mirrors the same catalog-honesty constraint
// buildingTypes.mappedCategorySlug documents: never suggests a product
// class (VRF, chiller, ducted central, industrial package units) elc
// doesn't actually sell.
func buildSuggestions(sig chatQuerySignals, totalCount int) []string {
	if sig.label == "" {
		// No confident category to deepen — nothing in the message to
		// anchor a "one more qualifier" suggestion to, so point at a few
		// concrete, distinct starting points instead of a vague follow-up
		// question with no context behind it.
		return []string{
			"Máy lạnh treo tường dưới 12 triệu",
			"Máy lọc nước RO 3 in 1 cho gia đình 4 người",
			"Máy cấp khí tươi lọc không khí giá tốt",
		}
	}

	context := sig.canonical
	if sig.hasSub {
		context += " " + sig.sub
	}
	if sig.buildingTypeName != "" {
		context += " cho " + sig.buildingTypeName
	}
	if sig.capacity.tier != "" {
		context += " " + sig.capacity.tier
	}

	var out []string
	// Form factor is the single biggest fork in elc's AC catalog (5
	// distinct categories) — worth asking about before finer qualifiers,
	// but only when the shopper hasn't already named one.
	if sig.label == "may_lanh" && !sig.hasSub {
		out = append(out, strings.TrimSpace(context+" âm trần"), strings.TrimSpace(context+" tủ đứng"))
	}
	if sig.label == "may_lanh" && sig.capacity.tier == "" {
		out = append(out, strings.TrimSpace(context+" cho phòng 20m2"))
	}
	if sig.minPrice == nil && sig.maxPrice == nil {
		out = append(out, strings.TrimSpace(context+" dưới 15 triệu"))
	}
	if sig.label == "may_lanh" && sig.gas == "" {
		out = append(out, strings.TrimSpace(context+" gas R32"))
	}
	if sig.sortBy == "" {
		if totalCount > ChatSearchLimit {
			out = append(out, strings.TrimSpace(context+" rẻ nhất"))
		} else {
			out = append(out, strings.TrimSpace(context+" mới nhất"))
		}
	}

	if len(out) > chatSuggestionLimit {
		out = out[:chatSuggestionLimit]
	}
	return out
}

// formatVNDTrieu renders a VND amount the way a shopper typed it back
// ("15 triệu", not "15,000,000") — matches parsePriceVND's own unit.
func formatVNDTrieu(v int64) string {
	trieu := float64(v) / 1_000_000
	if trieu == math.Trunc(trieu) {
		return strconv.FormatFloat(trieu, 'f', 0, 64) + " triệu"
	}
	return strconv.FormatFloat(trieu, 'f', 1, 64) + " triệu"
}

// formatNum trims a trailing ".0" the way a shopper would write a whole
// number ("2m", not "2.0m") while keeping one decimal place otherwise
// ("3.8m") — same Trunc-check idiom as formatVNDTrieu/formatHPTier.
func formatNum(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// capacityClause renders the reasoning behind capacity.tier — a distinct
// sentence shape per capacityKind, so the math is actually visible instead
// of just the final tier (a plain "công suất đề xuất 2 HP" hides *why*):
//   - direct HP: the shopper stated it outright, nothing to derive.
//   - dimension: shows the multiplication (dài × rộng = diện tích).
//   - volume: shows the m³→m² conversion this codebase's BTU rule uses.
//   - area: states the area as given.
//
// Heat-load-adjusted cases (dimension/volume/area only — a direct HP
// always wins outright, see applyCapacity) append a second clause showing
// the adjusted figure, not just the final tier, so "phòng 15m²... phù hợp
// 1.5 HP" doesn't silently become "2 HP" with no visible reason.
func capacityClause(c capacityInfo) string {
	switch c.kind {
	case capacityKindDirect:
		return "bạn yêu cầu trực tiếp công suất " + c.tier
	case capacityKindDimension:
		s := fmt.Sprintf("phòng dài %sm rộng %sm (diện tích %sm²)", formatNum(c.length), formatNum(c.width), formatNum(c.rawArea))
		return s + heatLoadClause(c) + ", phù hợp công suất " + c.tier
	case capacityKindVolume:
		s := fmt.Sprintf("thể tích phòng %sm³ (tương đương diện tích ~%sm²)", formatNum(c.volume), formatNum(c.rawArea))
		return s + heatLoadClause(c) + ", phù hợp công suất " + c.tier
	case capacityKindArea:
		s := fmt.Sprintf("phòng diện tích %sm²", formatNum(c.rawArea))
		return s + heatLoadClause(c) + ", phù hợp công suất " + c.tier
	default:
		return ""
	}
}

// heatLoadClause is the "(+X%, adjusted area Ym²)" fragment capacityClause
// appends when a heat-load multiplier was applied — "" otherwise, so the
// non-adjusted case doesn't get a dangling empty parenthetical.
func heatLoadClause(c capacityInfo) string {
	if c.heatLoadPercent == 0 {
		return ""
	}
	adjustedArea := c.rawArea * (1 + float64(c.heatLoadPercent)/100)
	if c.buildingTypeName != "" {
		return fmt.Sprintf(", cộng thêm %d%% do đặc thù tỏa nhiệt cao của %s (~%sm²)", c.heatLoadPercent, c.buildingTypeName, formatNum(adjustedArea))
	}
	return fmt.Sprintf(", cộng thêm %d%% do tải nhiệt cao hơn bình thường (~%sm²)", c.heatLoadPercent, formatNum(adjustedArea))
}

// stripStopwords drops common Vietnamese filler words ("tôi cần một cái
// ... cho ...") before handing the remainder to websearch_to_tsquery,
// which ANDs every bare word together — left unstripped, a full sentence
// would demand a product name containing "tôi" and "cần", matching
// nothing. Not a stemmer/tokenizer, just a denylist; good enough for
// typical shopper phrasing without pulling in an NLP dependency.
func stripStopwords(text string) string {
	words := strings.Fields(text)
	kept := words[:0]
	for _, w := range words {
		if _, isStop := vietnameseStopwords[strings.ToLower(w)]; !isStop {
			kept = append(kept, w)
		}
	}
	return strings.TrimSpace(strings.Join(kept, " "))
}

// minLeftoverWordRunes mirrors postgres_repository.go's minRunesForTrigram
// (same underlying reason, different mechanism): a short word is unreliable
// once matched *on its own* against a large free-text corpus, via
// immutable_unaccent collisions — e.g. "RO" (the water-filter abbreviation,
// a real query term) and "rõ" ("clearly", extremely common in marketing
// copy) both unaccent to "ro", so requiring just "ro" to appear anywhere in
// a máy lạnh product's description was enough to false-positive-match it
// for a "máy lọc nước RO" query.
const minLeftoverWordRunes = 4

// leftoverSearchTerm turns the marketing/feature words left over after
// structured-signal and stopword stripping into a Search-safe term:
//   - 2+ words: quoted as one adjacent phrase, e.g. "chống khô da" ->
//     `"chống khô da"`. This is the precise case — requiring the words
//     adjacent (rather than each independently, or dropping the short ones)
//     is what actually finds products describing that exact feature, not
//     just any product whose description happens to contain one of the
//     words somewhere. Short words inside a multi-word phrase are fine:
//     adjacency already rules out coincidental collisions like the "RO"/
//     "rõ" one below, since "rõ" surviving alone elsewhere in a long
//     description won't also have "khô" and "da" immediately next to it.
//   - exactly 1 word: only kept if it clears minLeftoverWordRunes — with no
//     neighboring words to require adjacency against, a short standalone
//     word has no such protection (see minLeftoverWordRunes).
func leftoverSearchTerm(text string) string {
	words := strings.Fields(text)
	switch len(words) {
	case 0:
		return ""
	case 1:
		if len([]rune(words[0])) < minLeftoverWordRunes {
			return ""
		}
		return words[0]
	default:
		return `"` + strings.Join(words, " ") + `"`
	}
}

var (
	// "từ 10 đến 20 triệu" / "10-20tr" — checked before under/over so its
	// "từ" doesn't get consumed by overPricePattern first.
	rangePricePattern = regexp.MustCompile(`(?i)từ\s*(\d+(?:[.,]\d+)?)\s*(?:đến|-|tới)\s*(\d+(?:[.,]\d+)?)\s*(?:tr|triệu)\b`)
	underPricePattern = regexp.MustCompile(`(?i)(?:dưới|duoi|không quá|khong qua|tối đa|toi da)\s*(\d+(?:[.,]\d+)?)\s*(?:tr|triệu)\b`)
	overPricePattern  = regexp.MustCompile(`(?i)(?:trên|tren|từ|tu|hơn|tối thiểu|toi thieu)\s*(\d+(?:[.,]\d+)?)\s*(?:tr|triệu)\b`)
	barePricePattern  = regexp.MustCompile(`(?i)(?:khoảng|khoang|tầm|tam)?\s*(\d+(?:[.,]\d+)?)\s*(?:tr|triệu)\b`)
)

type synonymRule struct {
	pattern   *regexp.Regexp
	canonical string
}

// categorySynonyms maps everyday phrasing to the term that actually
// appears in ELC's product names. First match wins (a message naming two
// categories is rare, and disambiguating that is out of scope for a
// rule-based parser). Fallback only — used when the classifier isn't
// available or isn't confident; see categoryLabelCanonical for its path.
var categorySynonyms = buildSynonymRules(map[string]string{
	"điều hòa|dieu hoa|máy điều hòa":    "máy lạnh",
	"lọc khí|loc khong khi|máy lọc khí": "lọc không khí",
	"lọc nước|loc nuoc":                 "lọc nước",
})

// categoryLabelCanonical maps a fastText prediction label (see
// cmd/train-chat-classifier's categoryTerms, which must stay in sync with
// these keys) to the search term used when the classifier is confident.
var categoryLabelCanonical = map[string]string{
	"may_lanh":       "máy lạnh",
	"loc_khong_khi":  "lọc không khí",
	"loc_nuoc":       "lọc nước",
	"nha_thong_minh": "nhà thông minh",
}

// knownBrands mirrors the real `brands` table (see `SELECT name FROM
// brands` — kept as a literal list rather than a live DB query since this
// package doesn't otherwise depend on the brand module, same reasoning as
// product's own CategoryRef/BrandRef read-only cross-module pattern).
// Update this list when a brand is added/removed. Aliases are alternate
// phrasings a shopper might use for that brand ("Mitsubishi Heavy" isn't a
// separate row in `brands`, just how people refer to Mitsubishi's AC
// line) — matched after normalizeVietnamese, so accents/case don't need
// listing separately. A slice (not a map) so match order — and therefore
// which brand wins if a message somehow names two — stays deterministic.
var knownBrands = []struct {
	canonical string
	aliases   []string
}{
	{"Acis", []string{"acis"}},
	{"Carrier", []string{"carrier"}},
	{"Daikin", []string{"daikin"}},
	{"Gree", []string{"gree"}},
	{"Hagisu", []string{"hagisu"}},
	{"LG", []string{"lg"}},
	{"Menred", []string{"menred"}},
	{"Midea", []string{"midea"}},
	{"Mitsubishi", []string{"mitsubishi", "mitsubishi heavy"}},
	{"Panasonic", []string{"panasonic"}},
	{"Samsung", []string{"samsung"}},
	{"Toshiba", []string{"toshiba"}},
}

type brandRule struct {
	pattern   *regexp.Regexp
	canonical string
}

var brandPatterns = buildBrandPatterns()

func buildBrandPatterns() []brandRule {
	var rules []brandRule
	for _, b := range knownBrands {
		for _, alias := range b.aliases {
			rules = append(rules, brandRule{
				pattern:   regexp.MustCompile(`\b` + regexp.QuoteMeta(normalizeVietnamese(alias)) + `\b`),
				canonical: b.canonical,
			})
		}
	}
	return rules
}

// detectBrand looks for a known brand name anywhere in text — matched
// against normalizeVietnamese(text) so typos in diacritics ("đaikin" for
// "Daikin") or missing accents don't miss it — and returns its canonical
// (correctly-capitalized) form. A lookup against the real brand list, not
// a guess: safe to append to Search even when the classifier already
// replaced everything else, since unlike generic leftover words it's
// guaranteed to appear in that brand's product names.
//
// Falls back to a fuzzy (edit-distance) match when no alias matches
// exactly — catches a dropped/swapped letter ("dakin" for "Daikin") that
// an exact regex would never match. Real gap found via testing: "mý lạnh
// dakin" fell through with no brand detected at all.
func detectBrand(text string) (string, bool) {
	normalized := normalizeVietnamese(text)
	for _, b := range brandPatterns {
		if b.pattern.MatchString(normalized) {
			return b.canonical, true
		}
	}
	for _, word := range strings.Fields(normalized) {
		if len([]rune(word)) < minFuzzyBrandRunes {
			continue
		}
		for _, b := range knownBrands {
			for _, alias := range b.aliases {
				aliasNorm := normalizeVietnamese(alias)
				// Multi-word aliases ("mitsubishi heavy") aren't a
				// single-word fuzzy target — the exact brandPatterns
				// check above already covers them.
				if strings.Contains(aliasNorm, " ") || len([]rune(aliasNorm)) < minFuzzyBrandRunes {
					continue
				}
				if levenshtein(word, aliasNorm) <= maxFuzzyBrandDistance {
					return b.canonical, true
				}
			}
		}
	}
	return "", false
}

// minFuzzyBrandRunes/maxFuzzyBrandDistance gate detectBrand's fuzzy
// fallback to longer brand names only (>=5 runes) — a distance-1 fuzzy
// match against a short name like "LG" or "Gree" would collide with all
// sorts of unrelated 2-4 letter words, trading a rare typo win for
// frequent false positives. Longer names have enough distinct characters
// that a 1-edit fuzzy match stays reliably brand-specific.
const (
	minFuzzyBrandRunes    = 5
	maxFuzzyBrandDistance = 1
)

// levenshtein computes the edit distance between two strings (rune-based,
// so Vietnamese multi-byte characters count as one edit each, not one per
// UTF-8 byte).
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = minInt(prev[j]+1, minInt(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

func minInt(a, b int) int {
	if b < a {
		return b
	}
	return a
}

func buildSynonymRules(m map[string]string) []synonymRule {
	rules := make([]synonymRule, 0, len(m))
	for pattern, canonical := range m {
		rules = append(rules, synonymRule{
			pattern:   regexp.MustCompile(`(?i)(?:` + pattern + `)`),
			canonical: canonical,
		})
	}
	return rules
}

// Every entry is a single word — stripStopwords checks word-by-word, so a
// multi-word phrase here would silently never match.
var vietnameseStopwords = func() map[string]struct{} {
	words := []string{
		"tôi", "toi", "mình", "minh", "tao", "em", "anh", "chị", "chi", "ạ", "a",
		"cần", "can", "muốn", "muon", "mua", "tìm", "tim", "kiếm", "kiem",
		"cho", "với", "voi", "và", "va", "là", "la", "có", "co",
		"một", "mot", "cái", "cai", "con", "để", "de", "được", "duoc",
		"nào", "nao", "gì", "gi", "nhé", "nhe", "giúp", "giup", "ơi", "oi",
		"xem", "shop", "bên", "ben", "bạn", "ban", "hãy", "hay",
		"làm", "ơn", "vui", "lòng", "long", "về", "ve", "của", "cua", "này", "nay",
		// Question/recommendation phrasing ("... nên dùng máy lạnh nào",
		// "... dùng loại gì cho tốt") — without these, leftover words like
		// "nên"/"dùng" never appear in a product name, so the AND-based
		// search_vector match returns nothing even when a structured
		// signal (price/attribute token) was extracted correctly.
		"nên", "nen", "dùng", "dung", "sử", "su", "dụng",
		"phù", "phu", "hợp", "hop", "tốt", "tot", "loại", "loai",
		"sao", "vậy", "vay", "thế", "the", "nhỉ", "nhi",
		"hoặc", "hoac",
		// Room-type words ("phòng khách 20m2", "phòng ngủ nhỏ") — generic
		// descriptors that never appear in a product's name field, same
		// reasoning as the question words above. "phòng" itself also
		// nearly always gets consumed by roomAreaPattern already, but not
		// when another word sits between it and the number ("phòng khách
		// 20m2"), so it needs to be caught here too.
		"phòng", "phong", "khách", "khach", "ngủ", "ngu", "bếp", "bep", "việc", "viec",
		// Install/spec-talk words ("nên lắp máy lạnh công suất nào") —
		// verified against real product names (none contain these), same
		// AND-pollution risk as everything else in this list. Found via
		// an actual zero-result query in production (phan_khuc_hp:3 HP
		// matched 17 real products, but "lắp"/"công suất" leftover in
		// Search zeroed it out anyway).
		"lắp", "lap", "đặt", "dat", "công", "cong", "suất", "suat",
		// Reverse-question / measurement-talk leftovers ("dùng cho phòng
		// rộng tối đa bao nhiêu mét vuông", "tổng thể tích 60m3 xài loại
		// nào") — same reasoning as the rest of this list: none of these
		// appear in a product name, and the number+unit they're attached
		// to is already consumed structurally by hpMentionPattern/
		// dimensionPattern/volumePattern/roomAreaPattern before this runs.
		"rộng", "rong", "tối", "toi", "đa", "da", "bao", "nhiêu", "nhieu",
		"mét", "met", "vuông", "vuong", "btu", "tổng", "tong", "thể", "the",
		"tích", "tich", "xài", "xai", "giá", "gia", "rẻ", "re",
		"tiết", "tiet", "kiệm", "kiem", "nhất", "nhat", "hiện", "hien",
		// "điện" ("tiết kiệm điện" = energy-saving) is a real word, not
		// filler — but it's a feature-description word that doesn't
		// appear in the product NAME field search_vector indexes, same
		// structural gap as "Comfort Air"/"Auto Clean"/etc (see
		// ChatSearchProducts' doc comment: description text isn't
		// searchable without a schema change). Treated as a stopword here
		// because leaving it in poisons the AND-match entirely rather
		// than just failing to add a filter.
		"điện", "dien",
		// NOT "không"/"khong": it's a filler question particle here
		// ("... nào không") but also the middle word of "lọc không khí"
		// (air purifier) — stripping it globally would corrupt that
		// canonical category term into "lọc khí", which matches nothing.
	}
	set := make(map[string]struct{}, len(words))
	for _, w := range words {
		set[w] = struct{}{}
	}
	return set
}()
