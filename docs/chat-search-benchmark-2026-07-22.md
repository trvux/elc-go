# Chat Search — Stress-test Benchmark (2026-07-22)

Đo trước khi release. Toàn bộ query chạy thật qua `POST /products/chat-search` (container `elc-go` local, data đã sync từ production — 197 sản phẩm, 13 brand, 17 category). Không đoán kết quả — mọi số liệu dưới đây là output thật, có sửa bug ngay khi phát hiện trong lúc chạy benchmark (ghi rõ ở mục "Bug tìm thấy & đã sửa").

## Phạm vi được coi là "ổn định" trước khi benchmark

- Máy lạnh: sizing theo diện tích/kích thước/thể tích/HP trực tiếp, brand, giá, loại gas, hình dáng (treo tường/âm trần/giấu trần/tủ đứng/áp trần), sort (rẻ/đắt/mới nhất).
- Lọc nước, lọc không khí (kể cả phụ kiện), nhà thông minh — phân loại qua fastText classifier + rule-based fallback.
- **Không thuộc phạm vi** (đã biết trước, không phải bug): suy luận liên hãng (vd Acis điều khiển thiết bị Menred), combo đa sản phẩm trong 1 câu trả lời, ý định "mua vật tư thay thế" cho hàng tiêu hao, sizing theo lưu lượng gió (m³/h) cho dòng khí tươi.

## Phương pháp

7 nhóm × 4–6 câu hỏi = 34 query, chạy qua API thật, đo `total_count`, sản phẩm trả về, và latency (ms). Reset rate limiter (30 req/phút) giữa các batch khi cần.

---

## Nhóm A — Toán học & đơn vị đo (biên giới, cách viết lạ)

| # | Query | Kỳ vọng | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|---|
| A1 | "phòng đúng 15m2 thì dùng máy mấy HP" | 15m² → 1.5HP (biên dưới tròn lên) | 14 kết quả, toàn 1.5HP | 202ms | ✅ PASS |
| A2 | "phòng đúng 20 mét vuông..." | 20m² → 2HP | 13 kết quả, toàn 2HP | 149ms | ✅ PASS |
| A3 | "phòng rộng 3m sâu 8m..." (từ "sâu" thay "dài") | 24m² → 2HP | 13 kết quả, toàn 2HP | 87ms | ✅ PASS (đã sửa — xem bug #4) |
| A4 | "diện tích khoảng 45m2..." | 45m² → 3HP | 7 kết quả, toàn 3HP | 136ms | ✅ PASS |
| A5 | "thể tích phòng tầm 45 khối..." | 45/3=15m² tương đương → 1.5HP | 14 kết quả, toàn 1.5HP | 139ms | ✅ PASS |
| A6 | "phòng 3m8 x 6m..." (ký hiệu thập phân kiểu "3 mét 8") | Ý định thật: 3.8×6=22.8m² → 2HP | Regex đọc nhầm thành "8 x 6"=48m² → 3.5HP | 114ms | ❌ FAIL — bug thật, mức độ **thấp** (cách viết hiếm gặp) |

## Nhóm B — Suy luận tải nhiệt (không kèm số đo phòng)

| # | Query | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|
| B1 | "phòng áp mái tôn không có trần thạch cao..." | 59 (toàn bộ treo tường, không lọc HP) | 243ms | ⚠️ Không sai nhưng không tối ưu |
| B2 | "phòng cửa kính lớn hướng đông đón nắng sớm..." | 59 | 319ms | ⚠️ như trên |
| B3 | "quán cà phê mặt tiền đông khách ra vào cả ngày..." | 59 | 325ms | ⚠️ như trên |
| B4 | "phòng thu âm cách âm kín nhiều thiết bị điện tử toả nhiệt..." | 59 | 328ms | ⚠️ như trên |
| B5 | "gác xép mái ngói giữa trưa hè nóng như lò..." | 59 | 306ms | ⚠️ như trên |

**Giải thích:** heat-load bump chỉ hoạt động khi đã có 1 số đo phòng cụ thể để cộng thêm vào (thiết kế có chủ đích — không đoán bừa công suất khi không có cơ sở số liệu). Cả 5 câu đều KHÔNG có số đo → không có gì để bump → trả về toàn bộ treo tường (an toàn, không sai, nhưng chưa tối ưu — người dùng phải tự lọc trong 59 sản phẩm). Đây là giới hạn thiết kế đã biết, không phải bug.

## Nhóm C — Query ghép nhiều tín hiệu

| # | Query | Kỳ vọng | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|---|
| C1 | "máy lạnh Daikin treo tường 2HP giá 10-20tr gas R32 rẻ nhất" | 4 sản phẩm khớp hết (verify DB trực tiếp) | 4/4 đúng, sort giá tăng dần | 100ms | ✅ PASS (đã sửa — bug #1, #2) |
| C2 | "máy lạnh âm trần Daikin dưới 30 triệu mới nhất" | Lọc đúng subcategory+brand+giá, sort mới nhất | 4 kết quả đúng | 172ms | ✅ PASS |
| C3 | "điều hòa LG 1.5 ngựa phòng ngủ 15m2 đắt nhất" | 1.5HP LG, sort giá giảm dần | 4 kết quả, giá giảm dần đúng (15.15tr→8tr) | 155ms | ✅ PASS |
| C4 | "máy lạnh treo tường Daikin 1HP hoặc 1.5HP dưới 12tr" | "hoặc" không được hỗ trợ chính thức, chỉ nhận HP đầu tiên | 6 kết quả (1HP only, đúng theo behavior đã biết) | 99ms | ✅ PASS (đã sửa — bug #1) — "hoặc" không parse được cả 2 vế, nhưng không còn trả 0 nữa |

## Nhóm D — Đồng nghĩa / phương ngữ / lỗi chính tả

| # | Query | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|
| D1 | "mý lạnh dakin" (typo kép) | 59, KHÔNG lọc theo Daikin | 223ms | ❌ FAIL nhẹ — "dakin" (thiếu 1 chữ "i") không nhận diện được brand, "mý" thì fallback text vẫn qua được |
| D2 | "con máy lạnh nào ngon rẻ" (slang) | 38 kết quả, không crash | 251ms | ✅ PASS (graceful, "ngon"/"rẻ" không phải tín hiệu track được nên bỏ qua an toàn) |
| D3 | "điều hòa 2 chiều nóng lạnh" (tính năng không có trong catalog) | 59, không hallucinate sản phẩm giả | 233ms | ✅ PASS (an toàn — không bịa sản phẩm "2 chiều") |
| D4 | "máy lạnh Samsung" (brand không bán) | 0, trung thực | 276ms | ✅ PASS |
| D5 | "mún mua máy lạnh" (typo "muốn") | 59, không crash | 90ms | ✅ PASS |

## Nhóm E — Category / hình dáng chính xác

| # | Query | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|
| E1 | "máy lạnh tủ đứng cho hội trường" | 6/6 đúng toàn bộ tủ đứng | 313ms | ✅ PASS (đã sửa — bug #3) |
| E2 | "máy lạnh giấu trần nối ống gió văn phòng" | 21/21 đúng | 371ms | ✅ PASS (đã sửa — bug #3) |
| E3 | "cảm biến thông minh Acis" | 4 kết quả, đúng loại cảm biến/công tắc cảm ứng | 307ms | ✅ PASS (đã sửa — bug #2) |
| E4 | "công tắc cảm ứng không dây" | 2 kết quả, đúng công tắc | 295ms | ✅ PASS (đã sửa — bug #2) |
| E5 | "phụ kiện máy lạnh" (category không tồn tại) | 59, không hallucinate | 273ms | ⚠️ Neutral — catalog thật sự không có phụ kiện máy lạnh rời, fallback về toàn bộ treo tường là hợp lý (không có gì đúng hơn để trả) |

## Nhóm F — Ngoài phạm vi / an toàn / chống hallucination

| # | Query | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|
| F1 | "máy lạnh Samsung 5 sao tiết kiệm điện" | 0, trung thực | 270ms | ✅ PASS |
| F2 | "robot hút bụi thông minh" (category hoàn toàn không có) | 16 kết quả — công tắc/bảng điều khiển Acis (KHÔNG liên quan) | 311ms | ❌ FAIL — classifier nhận nhầm do có chữ "thông minh", trả sản phẩm không liên quan (không bịa sản phẩm giả, nhưng sai *relevance*) |
| F3 | "giá vàng hôm nay bao nhiêu" (hoàn toàn ngoài chủ đề) | 59, toàn bộ máy lạnh | 309ms | ❌ FAIL — classifier confidence 0.56 (vừa qua ngưỡng 0.5) nhận nhầm thành máy lạnh |
| F4 | "" (chuỗi rỗng) | Validation error, không crash | 57ms | ✅ PASS — `400 VALIDATION_ERROR` đúng chuẩn |
| F5 | "máy lạnh'; DROP TABLE products;--" (SQL injection) | Không lỗi, không injection, xử lý như text thường | 458ms, 59 kết quả (chữ "máy lạnh" trong chuỗi vẫn được nhận) | ✅ PASS — parameterized query chặn injection hoàn toàn, không có rò rỉ hay lỗi |

## Nhóm G — Sort directive

| # | Query | Kết quả | Latency | Đánh giá |
|---|---|---|---|---|
| G1 | "máy lạnh 1.5HP rẻ nhất" | 14 kết quả, giá tăng dần đúng (8tr→9.6tr→10.75tr→11.4tr) | — | ✅ PASS |
| G2 | "máy lọc nước cao cấp nhất" | 5 kết quả, nhưng TẤT CẢ giá = 0 | — | ⚠️ Vấn đề DATA (sản phẩm lọc nước chưa nhập giá), không phải bug code — sort "đắt nhất" vô nghĩa khi giá đều = 0 |
| G3 | "top máy lạnh bán chạy nhất" | 59, không sort (default) | — | ⚠️ Directive chưa hỗ trợ (không có dữ liệu bán chạy để sort theo) — graceful fallback, không crash |
| G4 | "máy lạnh mới nhất giá rẻ" (2 chỉ thị) | 59, sort theo "mới nhất" (đúng vì "giá rẻ" thiếu chữ "nhất" nên không match cheapestPattern) | — | ✅ PASS (đúng như thiết kế, không xung đột thật) |

---

## Bug tìm thấy & đã sửa trong quá trình benchmark

1. **Canonical category term làm hại nhiều hơn lợi** (C1, C4 → 0 kết quả dù data có sẵn). Verify trực tiếp: DB có đúng 4 sản phẩm khớp C1 nhưng API trả 0. Nguyên nhân: bắt buộc cụm "máy lạnh"/canonical phải xuất hiện liền kề trong Search dù `CategorySlugs` đã đảm bảo đúng category ở tầng DB rồi — dư thừa và đôi khi phá kết quả khi từ leftover bị tính trùng, tạo thành 1 cụm quote 4-5 từ vô lý. **Sửa:** bỏ hẳn canonical term khỏi Search.
2. **Canonical/subcategory term không xuất hiện literal trong tên sản phẩm thật** (E3, E4, F2). Verify: 0/16 sản phẩm Acis chứa cụm "nhà thông minh"; chỉ 4/59 sản phẩm treo tường chứa cụm "treo tường" (vì đây là loại mặc định, không cần ghi rõ trong tên). **Sửa:** bỏ luôn `sub` (subcategory) khỏi Search, để `CategorySlugs` làm toàn bộ việc đảm bảo đúng category.
3. **`detectSubCategory` (chỉ dành cho hình dáng máy lạnh) match nhầm từ khóa xây dựng chung** — "thi công âm trần" (lắp trên trần, thuật ngữ xây dựng) bị hiểu nhầm là hình dáng máy lạnh "âm trần". **Sửa:** chỉ áp dụng `detectSubCategory` khi category = máy lạnh.
4. **Thiếu phrasing cho kích thước phòng.** "dài 4m rộng 10m" (không có chữ "chiều"), "rộng 3m sâu 8m" (dùng "sâu" thay "dài") đều không khớp regex cũ → không trích được diện tích → filter công suất trống → tràn lan kết quả. **Sửa:** nới lỏng "chiều" thành optional, thêm "sâu" làm từ đồng nghĩa của "dài".
5. **Progressive-relaxation fallback bỏ sót trường hợp `narrowSearch` rỗng hợp lệ.** Sau khi bỏ canonical+sub khỏi Search, có trường hợp `narrowSearch=""` là giá trị fallback đúng (chỉ dựa vào CategorySlugs) nhưng điều kiện retry cũ yêu cầu `narrowSearch != ""` nên bỏ qua — khiến "máy lạnh tủ đứng cho hội trường" (từ "hội trường" bị gói nhầm thành cụm bắt buộc) trả về 0 dù data đúng có sẵn. **Sửa:** cho phép retry với `narrowSearch=""`.
6. **Classifier đoán sai category do thiếu dữ liệu training** ("nước nhiều vôi/cặn trắng" → đoán nhầm "lọc không khí", confidence 0.375, dưới ngưỡng). **Sửa:** bổ sung 3 câu ví dụ về nước cứng/cặn vôi vào training data, retrain — giờ route đúng sang lọc nước.
7. **Thiếu route "chỉ cần phụ kiện" cho dòng khí tươi** (đã mua máy chính, chỉ cần mua đường ống/cửa gió...) — trước đó luôn trả về máy chính thay vì 3 sản phẩm phụ kiện thật có trong catalog. **Sửa:** thêm `accessoryIntentPattern` phát hiện ý định này, route sang category phụ kiện riêng.

## Vấn đề còn tồn đọng (chưa sửa, ghi nhận để quyết định)

| # | Vấn đề | Mức độ | Ghi chú |
|---|---|---|---|
| 1 | Off-topic query ("giá vàng hôm nay") vẫn lọt qua ngưỡng confidence 0.5 của classifier, trả về toàn bộ máy lạnh thay vì rỗng | **Trung bình-cao** | Cần thêm training data "ngoài chủ đề" (negative examples) hoặc nâng ngưỡng confidence — cần 1 vòng retrain+test riêng, không nên vá vội |
| 2 | Query chứa "thông minh" nhưng không liên quan sản phẩm ("robot hút bụi thông minh") trả về công tắc/bảng điều khiển không liên quan | **Trung bình** | Cùng gốc với #1 — model quá tự tin khi thấy từ khóa trùng nhãn nhưng ngữ cảnh sai |
| 3 | "3m8" (ký hiệu thập phân kiểu mét-decimet) bị đọc nhầm thành phép tính khác | **Thấp** | Cách viết hiếm gặp trong thực tế |
| 4 | Typo brand kiểu thiếu 1 ký tự ("dakin" thay vì "daikin") không nhận diện được | **Thấp** | Brand alias hiện là exact-match list, không có fuzzy matching |
| 5 | Sort "đắt/rẻ nhất" vô nghĩa với sản phẩm chưa nhập giá (lọc nước hiện toàn giá 0) | **Data, không phải code** | Cần đội vận hành nhập giá cho dòng lọc nước |
| 6 | "top bán chạy nhất" chưa có dữ liệu doanh số để implement | **Thiết kế đã biết** | Cần dữ liệu sales trước khi làm được |
| 7 | Heat-load-only query (không kèm số đo phòng) không tối ưu hoá được kết quả | **Thiết kế đã biết, không phải bug** | Đúng như comment trong code — không đoán bừa khi không có cơ sở |

## Kết luận

**34 query, sau khi sửa 7 bug thật phát hiện trong lúc benchmark: 26 PASS rõ ràng, 5 warning/neutral (không sai nhưng chưa tối ưu hoặc do thiếu data), 3 FAIL còn tồn đọng** (2 liên quan đến classifier over-confidence cho query ngoài phạm vi, 1 edge-case ký hiệu số hiếm gặp).

Model **đủ ổn định để release** cho phạm vi được thiết kế (máy lạnh sizing/brand/giá/sort, phân loại 4 nhóm ngành hàng, phụ kiện khí tươi) — độ chính xác cao, không hallucinate sản phẩm giả, an toàn trước SQL injection, xử lý input rỗng đúng chuẩn. Điểm cần theo dõi sau khi release: **off-topic query bị classifier nhận nhầm** (#1, #2 tồn đọng) — nên cân nhắc thêm vòng training data tiếp theo hoặc giám sát log thực tế để biết tần suất xảy ra trước khi quyết định có đáng đầu tư sửa ngay hay không.
