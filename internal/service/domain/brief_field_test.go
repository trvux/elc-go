package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestBriefKeyFromLabel(t *testing.T) {
	cases := map[string]string{
		"Số phòng cần làm lạnh":       "so_phong_can_lam_lanh",
		"Diện tích (m²)":              "dien_tich_m",
		"Đã có bản vẽ mặt bằng chưa?": "da_co_ban_ve_mat_bang_chua",
		"  Loại   công trình  ":       "loai_cong_trinh",
		"???":                         "",
	}
	for label, want := range cases {
		if got := BriefKeyFromLabel(label); got != want {
			t.Errorf("BriefKeyFromLabel(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestCleanBriefFields_DefaultsKindAndDerivesKey(t *testing.T) {
	got, err := CleanBriefFields([]BriefField{{Label: " Số tầng ", Kind: "number", Unit: " tầng "}, {Label: "Ghi chú"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []BriefField{
		{Key: "so_tang", Label: "Số tầng", Kind: BriefKindNumber, Unit: "tầng"},
		{Key: "ghi_chu", Label: "Ghi chú", Kind: BriefKindText},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestCleanBriefFields_DropsBlankLabelsAndReturnsNonNil(t *testing.T) {
	got, err := CleanBriefFields([]BriefField{{Label: "  "}, {}})
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("got %#v, %v; want empty non-nil slice and no error", got, err)
	}
}

func TestCleanBriefFields_KeepsAnExplicitKeyWhenTheLabelChanges(t *testing.T) {
	got, _ := CleanBriefFields([]BriefField{{Key: "area_m2", Label: "Diện tích cần làm lạnh", Kind: BriefKindArea}})
	if got[0].Key != "area_m2" {
		t.Errorf("key = %q, want the stored key kept", got[0].Key)
	}
}

func TestCleanBriefFields_KeysStayUniqueWithinAService(t *testing.T) {
	got, _ := CleanBriefFields([]BriefField{{Label: "Địa chỉ"}, {Label: "Dia chi"}, {Key: "dia_chi", Label: "Khác"}})
	keys := []string{got[0].Key, got[1].Key, got[2].Key}
	if !reflect.DeepEqual(keys, []string{"dia_chi", "dia_chi_2", "dia_chi_3"}) {
		t.Errorf("keys = %v", keys)
	}
}

func TestCleanBriefFields_Rejections(t *testing.T) {
	long := strings.Repeat("a", maxBriefLabelRunes+1)
	cases := map[string][]BriefField{
		"unknown kind":           {{Label: "X", Kind: "slider"}},
		"choice with one option": {{Label: "X", Kind: BriefKindChoice, Options: []string{"Chỉ một", " "}}},
		"choice with no options": {{Label: "X", Kind: BriefKindChoice}},
		"bad explicit key":       {{Key: "Có Dấu", Label: "X"}},
		"label too long":         {{Label: long}},
		"too many options":       {{Label: "X", Kind: BriefKindChoice, Options: manyOptions(maxBriefOptions + 1)}},
		"too many fields":        manyFields(maxBriefFields + 1),
	}
	for name, in := range cases {
		if _, err := CleanBriefFields(in); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}

func TestCleanBriefFields_ChoiceOptionsAreTrimmedAndDeduped(t *testing.T) {
	got, err := CleanBriefFields([]BriefField{{Label: "Loại", Kind: BriefKindChoice, Options: []string{" Nhà phố ", "nhà phố", "Biệt thự", ""}}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got[0].Options, []string{"Nhà phố", "Biệt thự"}) {
		t.Errorf("options = %v", got[0].Options)
	}
}

func TestCleanBriefFields_UnitAndOptionsOnlyKeptWhereTheyApply(t *testing.T) {
	got, _ := CleanBriefFields([]BriefField{{Label: "A", Kind: BriefKindText, Unit: "x", Options: []string{"a", "b"}}})
	if got[0].Unit != "" || got[0].Options != nil {
		t.Errorf("text field kept unit/options: %#v", got[0])
	}
}

func TestNewService_BriefFieldProblemsAreReportedWithTheOtherFields(t *testing.T) {
	_, err := NewService("", "slug", "", nil, LineDichVu, nil, []BriefField{{Label: "X", Kind: "slider"}}, nil, nil, nil, nil, nil, true, 0)
	if err == nil || !strings.Contains(err.Error(), "validation") {
		t.Fatalf("expected a validation error, got %v", err)
	}
}

func TestService_Update_BriefFields(t *testing.T) {
	s, err := NewService("T", "slug", "", nil, LineThiCong, nil, []BriefField{{Label: "A"}}, nil, nil, nil, nil, nil, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(UpdateServiceInput{}); err != nil || len(s.BriefFields()) != 1 {
		t.Fatalf("an unrelated update changed the fields: %v %v", err, s.BriefFields())
	}
	if err := s.Update(UpdateServiceInput{BriefFields: []BriefField{{Label: "B", Kind: BriefKindYesNo}}}); err != nil {
		t.Fatal(err)
	}
	if got := s.BriefFields(); len(got) != 1 || got[0].Key != "b" || got[0].Kind != BriefKindYesNo {
		t.Fatalf("fields = %#v", got)
	}
	if err := s.Update(UpdateServiceInput{BriefFields: []BriefField{{Label: "C", Kind: "slider"}}}); err == nil {
		t.Fatal("an invalid field must fail the update")
	}
	if len(s.BriefFields()) != 1 || s.BriefFields()[0].Label != "B" {
		t.Fatalf("a failed update must leave the fields alone, got %#v", s.BriefFields())
	}
	if err := s.Update(UpdateServiceInput{BriefFields: []BriefField{}}); err != nil || len(s.BriefFields()) != 0 {
		t.Fatalf("an empty list must clear: %v %#v", err, s.BriefFields())
	}
}

func manyOptions(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "opt" + itoa(i)
	}
	return out
}

func manyFields(n int) []BriefField {
	out := make([]BriefField, n)
	for i := range out {
		out[i] = BriefField{Label: "Field " + itoa(i)}
	}
	return out
}
