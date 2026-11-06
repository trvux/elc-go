package domain

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// BriefField is one thing the customer is asked for before staff can quote: a
// typed replacement for the free-text brief_prompts. The typed kind lets the lead
// form render the right control (chips instead of a text box), offer "Chưa biết"
// per field, and store a stable key, so a lead's answer keeps its meaning when
// admin later rewords the label.
type BriefField struct {
	// Key is the stable identifier a lead's answer is stored under ("area_m2").
	// Empty on input means "derive it from the label".
	Key   string `json:"key"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
	// Unit is shown after a number field ("phòng", "tầng"). number only.
	Unit string `json:"unit,omitempty"`
	// Options are the chips of a choice field. choice only; the area kind has
	// fixed bands defined in the lead form, so it takes none.
	Options []string `json:"options,omitempty"`
}

const (
	BriefKindText   = "text"   // one line of free text
	BriefKindNumber = "number" // a small count, with an optional unit
	BriefKindChoice = "choice" // pick one of Options
	BriefKindYesNo  = "yesno"  // Có / Không
	BriefKindArea   = "area"   // floor area to cool, answered as a band (m²)
)

const (
	maxBriefFields       = 12
	maxBriefLabelRunes   = 120
	maxBriefKeyRunes     = 40
	maxBriefUnitRunes    = 20
	maxBriefOptions      = 12
	maxBriefOptionRunes  = 60
	minBriefChoiceOption = 2
)

func validBriefKind(kind string) bool {
	switch kind {
	case BriefKindText, BriefKindNumber, BriefKindChoice, BriefKindYesNo, BriefKindArea:
		return true
	}
	return false
}

// CleanBriefFields trims, defaults and validates admin input. It returns a
// non-nil slice (the column is NOT NULL). Blank-label rows are dropped like the
// blank lines of the old textarea; everything else wrong is reported, because
// silently repairing a typed field (say, a choice with one option) would hide a
// mistake that the customer would then see.
func CleanBriefFields(in []BriefField) ([]BriefField, error) {
	out, problems := cleanBriefFields(in)
	if len(problems) > 0 {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"briefFields": problems})
	}
	return out, nil
}

// cleanBriefFields is CleanBriefFields with the problems as a list, so NewService
// can fold them into its own field-error map.
func cleanBriefFields(in []BriefField) ([]BriefField, []string) {
	out := make([]BriefField, 0, len(in))
	used := map[string]bool{}
	problems := []string{}

	for _, f := range in {
		label := strings.TrimSpace(f.Label)
		if label == "" {
			continue
		}
		n := len(out) + 1
		if len([]rune(label)) > maxBriefLabelRunes {
			problems = append(problems, labelProblem(n, "label is too long"))
			continue
		}

		kind := strings.TrimSpace(f.Kind)
		if kind == "" {
			kind = BriefKindText
		}
		if !validBriefKind(kind) {
			problems = append(problems, labelProblem(n, "unknown kind "+kind))
			continue
		}

		key := strings.TrimSpace(f.Key)
		if key == "" {
			key = BriefKeyFromLabel(label)
		}
		if key == "" || len([]rune(key)) > maxBriefKeyRunes || !validBriefKey(key) {
			problems = append(problems, labelProblem(n, "key must be lowercase letters, digits and _ (max 40)"))
			continue
		}
		key = uniqueBriefKey(key, used)

		clean := BriefField{Key: key, Label: label, Kind: kind}
		switch kind {
		case BriefKindNumber:
			clean.Unit = strings.TrimSpace(f.Unit)
			if len([]rune(clean.Unit)) > maxBriefUnitRunes {
				problems = append(problems, labelProblem(n, "unit is too long"))
				continue
			}
		case BriefKindChoice:
			opts, err := cleanOptions(f.Options)
			if err != "" {
				problems = append(problems, labelProblem(n, err))
				continue
			}
			clean.Options = opts
		}
		out = append(out, clean)
	}

	if len(out) > maxBriefFields {
		problems = append(problems, "at most 12 fields")
	}
	return out, problems
}

func labelProblem(n int, msg string) string {
	return "field " + itoa(n) + ": " + msg
}

func itoa(n int) string { return strconv.Itoa(n) }

func cleanOptions(in []string) ([]string, string) {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, o := range in {
		o = strings.TrimSpace(o)
		if o == "" || seen[strings.ToLower(o)] {
			continue
		}
		if len([]rune(o)) > maxBriefOptionRunes {
			return nil, "an option is too long"
		}
		seen[strings.ToLower(o)] = true
		out = append(out, o)
	}
	if len(out) < minBriefChoiceOption {
		return nil, "a choice needs at least 2 options"
	}
	if len(out) > maxBriefOptions {
		return nil, "a choice has at most 12 options"
	}
	return out, ""
}

func validBriefKey(key string) bool {
	for _, r := range key {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
	}
	return true
}

// uniqueBriefKey keeps keys unique inside one service ("dia_chi", "dia_chi_2").
func uniqueBriefKey(key string, used map[string]bool) string {
	candidate := key
	for n := 2; used[candidate]; n++ {
		candidate = key + "_" + itoa(n)
	}
	used[candidate] = true
	return candidate
}

// BriefKeyFromLabel makes a stable-looking key from a Vietnamese label: strip
// diacritics, lowercase, non-alphanumerics to "_". "Số phòng cần làm lạnh" ->
// "so_phong_can_lam_lanh". Only used when admin leaves the key empty; after that
// the stored key is kept even if the label changes.
func BriefKeyFromLabel(label string) string {
	label = strings.ReplaceAll(strings.ReplaceAll(label, "đ", "d"), "Đ", "d")
	var b strings.Builder
	lastUnderscore := true
	for _, r := range norm.NFD.String(strings.ToLower(label)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	key := strings.Trim(b.String(), "_")
	if r := []rune(key); len(r) > maxBriefKeyRunes {
		key = strings.TrimRight(string(r[:maxBriefKeyRunes]), "_")
	}
	return key
}
