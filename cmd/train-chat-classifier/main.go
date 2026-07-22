// train-chat-classifier generates fastText training data for the chat
// search category classifier from a lookup table (terms/synonyms per
// category), instead of hand-writing hundreds of example sentences —
// every {term} × {template} combination becomes one training line. Run
// via `go run ./cmd/train-chat-classifier > train.txt`, then train with
// the fasttext CLI (see internal/product/application/chat_classifier.go
// for how the resulting model is used at runtime).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// categoryTerms is the lookup table: fastText label -> every phrasing a
// shopper might use for that category, sourced from the real category
// names in the DB (group_categories) plus everyday synonyms. Extend this
// when a new category group is added — the generator does the rest.
var categoryTerms = map[string][]string{
	"may_lanh": {
		"máy lạnh", "điều hòa", "máy điều hòa", "máy lạnh treo tường",
		"máy lạnh âm trần", "máy lạnh tủ đứng", "máy lạnh áp trần",
		"máy lạnh multi", "AC",
	},
	"loc_khong_khi": {
		"máy lọc không khí", "lọc không khí", "máy cấp khí tươi",
		"cấp khí tươi", "máy lọc khí",
	},
	"loc_nuoc": {
		"máy lọc nước", "lọc nước", "máy lọc nước RO", "nước RO", "máy RO",
	},
	"nha_thong_minh": {
		"nhà thông minh", "smart home", "công tắc thông minh",
		"cảm biến thông minh", "remote cầm tay", "bảng điều khiển thông minh",
		"thiết bị thông minh",
	},
}

// templates use {term} as the substitution point. Deliberately generic
// (not category-tailored) — fastText's bag-of-n-grams model mostly cares
// about which distinguishing words co-occur, not sentence naturalness, so
// a slightly odd combination ("nhà thông minh dưới 10 triệu") still
// teaches it the right thing.
var templates = []string{
	"{term}",
	"tôi cần {term}",
	"tôi muốn mua {term}",
	"shop có {term} không",
	"{term} giá rẻ",
	"{term} giá tốt",
	"{term} chính hãng",
	"tư vấn {term}",
	"nên mua {term} nào",
	"{term} loại nào tốt",
	"cho tôi xem {term}",
	"{term} cho gia đình",
	"{term} dưới 10 triệu",
	"{term} khoảng 15 triệu",
	"{term} trên 5 triệu",
	"mua {term} ở đâu uy tín",
	"{term} nào đang bán chạy",
}

// narrativeExamples are full, longer sentences (not template-generated)
// closer to how shoppers actually describe a situation rather than name a
// category outright — the template × term combinations above are all
// short and fairly uniform in structure, so a longer narrative message
// measured lower classifier confidence (0.58-0.60, still correct, just
// under a stricter threshold) despite having no closer match in the
// training set to draw on. These don't need to be exhaustive, just give
// the model more exposure to that sentence shape per category.
var narrativeExamples = map[string][]string{
	"may_lanh": {
		"nhà trọ gác xép cao tổng thể tích 60m3 thì xài loại nào",
		"phòng 15m2 nhưng tường gạch mỏng bị nắng chiếu trực tiếp cả ngày hướng tây",
		"quán bida quán net đông người ra vào mở cửa liên tục nên lắp loại nào",
		"phòng ăn nối liền bếp nấu nướng rộng chọn máy lạnh sao cho mát",
		"lắp máy lạnh cho nhà kho chứa hàng phòng server chạy 24/7",
		"chung cư trần thạch cao thấp có lắp được máy lạnh âm trần cassette không",
		"nhà thuê không cho khoan tường thì dùng máy lạnh gì",
		"phòng tập gym phòng yoga cần làm lạnh nhanh",
		// Short building-type + sort-directive phrasing — found via
		// testing that this specific shape ("[venue] [size] [superlative]",
		// no explicit "máy lạnh"/"điều hòa" word) scored under
		// chatClassifierMinConfidence and got misclassified as "khac",
		// even though longer/more descriptive versions of the same intent
		// (e.g. "nhà hàng buffet 20m2 lắp máy lạnh gì") classify correctly.
		"nhà hàng 20m2 rẻ nhất",
		"quán cafe 25m2 mắc nhất",
		"biệt thự nên lắp máy lạnh Daikin nào",
		"văn phòng 30m2 mới nhất",
	},
	"loc_nuoc": {
		"nước máy nhà tôi bị đục cần lọc",
		"nước giếng nhà tôi có mùi tanh muốn xử lý sạch để uống được",
		"gia đình đông người cần hệ thống lọc nước sinh hoạt cho cả nhà",
		"nước nhà tôi nhiều vôi bị đóng cặn trắng ở bình đun cần xử lý",
		"nước cứng đóng cặn ở vòi sen ấm siêu tốc muốn lọc sạch toàn nhà",
		"bình nóng lạnh nhà tôi hay bị đóng cặn vôi cần lọc nước đầu nguồn",
	},
	"loc_khong_khi": {
		"không khí trong nhà hơi bí muốn cải thiện",
		"nhà gần đường lớn nhiều bụi mịn cần thiết bị lọc sạch không khí",
		"trẻ nhỏ hay bị dị ứng nên cần làm sạch không khí trong phòng",
	},
	"nha_thong_minh": {
		"muốn điều khiển đèn quạt trong nhà bằng điện thoại từ xa",
		"nhà đang xây muốn lắp hệ thống điều khiển tự động hiện đại",
	},
	// "khac" (out-of-domain) exists purely so the softmax has somewhere
	// to route off-topic text instead of being forced to pick one of the
	// 4 real categories — fastText's predict-prob always returns SOME
	// label with SOME confidence, so without negative examples an
	// unrelated query ("giá vàng hôm nay bao nhiêu") can accidentally
	// cross chatClassifierMinConfidence for whichever real label its
	// vocabulary happens to overlap with (verified in testing: "robot
	// hút bụi thông minh" hit nha_thong_minh at 0.9999 confidence purely
	// because it contains "thông minh", then chat search returned
	// unrelated switch/sensor products). No code change needed for this:
	// categoryLabelCanonical (chat_search.go) has no "khac" key, so
	// parseChatQuery's `if canonical, ok := categoryLabelCanonical[label]`
	// check already falls through to the regex/synonym fallback path for
	// any label it doesn't recognize.
	"khac": {
		"giá vàng hôm nay bao nhiêu",
		"giá đô la hôm nay thế nào",
		"thời tiết hôm nay có mưa không",
		"robot hút bụi thông minh",
		"điện thoại iphone mới nhất giá bao nhiêu",
		"tivi samsung 55 inch giá bao nhiêu",
		"xe máy honda mới nhất",
		"quần áo trẻ em giá rẻ",
		"đặt vé máy bay đi hà nội",
		"trận bóng đá tối nay mấy giờ",
		"cách nấu phở bò ngon",
		"tỷ giá đô hôm nay",
		"tin tức mới nhất hôm nay",
		"quán ăn ngon gần đây",
		"lịch chiếu phim rạp cgv",
		"cách giảm cân nhanh",
		"laptop gaming giá rẻ",
		"tủ lạnh samsung giá bao nhiêu",
		"xin chào bạn khỏe không",
		"cảm ơn bạn nhiều",
	},
}

func main() {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for label, terms := range categoryTerms {
		for _, term := range terms {
			for _, tmpl := range templates {
				sentence := strings.ReplaceAll(tmpl, "{term}", term)
				fmt.Fprintf(w, "__label__%s %s\n", label, sentence)
			}
		}
	}

	for label, sentences := range narrativeExamples {
		for _, sentence := range sentences {
			fmt.Fprintf(w, "__label__%s %s\n", label, sentence)
		}
	}
}
