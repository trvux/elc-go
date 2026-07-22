# Chat Search — Building Taxonomy Benchmark (2026-07-22)

Test thật cho tính năng building taxonomy (loại công trình → hệ số tải nhiệt + category phù hợp) vừa thêm vào chat search. Chạy 2 lượt qua `POST /products/chat-search` (container `elc-go` local, data sync từ production) — lượt 1 phát hiện 3 bug, đã sửa, lượt 2 xác nhận toàn bộ 26/26 đúng và ổn định.

## Thiết kế đã áp dụng

- **14 loại công trình / 6 nhóm** từ taxonomy input, mỗi loại có `heat_load_factor` riêng thay cho hệ số chung 1.25 trước đây.
- **Chỉ map category cho loại công trình catalog thật đáp ứng được**: biệt thự/khách sạn/resort → giấu trần nối ống gió; quán cafe/nhà hàng/showroom/văn phòng → âm trần cassette; nhà phố → treo tường.
- **Không ép category cho loại công trình catalog không có hàng thật**: bệnh viện, phòng sạch, data center/phòng server, nhà xưởng công nghiệp, hội trường/nhà thờ trần siêu cao — vẫn tính `heat_load_factor` đúng (để không under-size), nhưng category rơi về mặc định máy lạnh dân dụng thay vì giả vờ có VRV/chiller/AHU/phòng sạch. Đây là chủ đích, không phải thiếu sót — tránh gợi ý sai (1 máy lạnh split không thể thay thế hệ thống trung tâm).
- **User tự nói rõ hình dáng máy → luôn thắng** building-type default (đã verify ở E1).

## Kết quả (lượt cuối, sau khi sửa hết bug)

### Nhóm A — Dân dụng cao cấp/đô thị

| # | Query | Diện tích × heat_load_factor | Kỳ vọng | Kết quả | Đánh giá |
|---|---|---|---|---|---|
| A1 | "biệt thự 20m2 cần lắp máy lạnh gì" | 20×1.2=24m² → 2HP | Category=giấu trần, 2HP | 3 kết quả, đúng giấu trần 2HP | ✅ |
| A2 | "penthouse 25m2..." (alt keyword) | 25×1.2=30m² → 2.5HP | Category=giấu trần, 2.5HP | 3 kết quả, đúng | ✅ |
| A3 | "chung cư cao cấp 20m2..." | 20×1.0=20m² → 2HP | Unmapped → fallback treo tường mặc định | 13 kết quả (= baseline treo tường 2HP) | ✅ Honest fallback đúng thiết kế |
| A4 | "nhà phố 20m2..." | 20×1.15=23m² → 2HP | Category=treo tường (trùng mặc định) | 13 kết quả | ✅ |
| A5 | "nhà ống hẹp 15m2..." (alt keyword) | 15×1.15=17.25m² → 1.5HP | Category=treo tường, 1.5HP | 14 kết quả | ✅ |

### Nhóm B — Thương mại mật độ cao

| # | Query | heat_load_factor | Kết quả | Đánh giá |
|---|---|---|---|---|
| B1 | "quán cafe 20m2..." | 1.3x → 26m²→2HP | 3 kết quả, âm trần 2HP | ✅ |
| B2 | "quán trà sữa 18m2..." (alt keyword) | 1.3x → 23.4m²→2HP | 3 kết quả, âm trần 2HP | ✅ |
| B3 | "nhà hàng buffet 20m2..." | 1.4x → 28m²→2HP | 3 kết quả, âm trần 2HP | ✅ |
| B4 | "showroom quần áo 25m2..." | 1.25x → 31.25m²→2.5HP | 3 kết quả, âm trần 2.5HP | ✅ |
| B5 | "boutique thời trang 20m2..." (alt keyword) | 1.25x → 25m²→2HP | 3 kết quả, âm trần 2HP | ✅ |

### Nhóm C — Loại công trình KHÔNG có hàng thật (test honest fallback)

| # | Query | heat_load_factor | Category | Kết quả | Đánh giá |
|---|---|---|---|---|---|
| C1 | "hội trường 20m2..." | 1.5x → 30m²→2.5HP | Không ép (treo tường mặc định) | 11 kết quả, đúng tier 2.5HP | ✅ Honest, capacity đúng |
| C2 | "nhà thờ 20m2..." | 1.3x → 26m²→2HP | Không ép | 13 kết quả | ✅ |
| C3 | "bệnh viện 20m2..." | 1.2x → 24m²→2HP | Không ép | 13 kết quả | ✅ |
| C4 | "phòng sạch lab 20m2..." | 1.4x → 28m²→2HP | Không ép | 13 kết quả | ✅ |
| C5 | "phòng server 20m2..." | 1.8x (cao nhất) → 36m²→2.5HP | Không ép | 11 kết quả, tier nâng đúng | ✅ |
| C6 | "nhà xưởng sản xuất 20m2..." | 1.5x → 30m²→2.5HP | Không ép | 11 kết quả | ✅ (đã sửa — xem bug #3) |

### Nhóm D — Văn phòng & lưu trú

| # | Query | Category | Kết quả | Đánh giá |
|---|---|---|---|---|
| D1 | "văn phòng 20m2..." | âm trần | 3 kết quả, 2HP | ✅ |
| D2 | "khách sạn 20m2..." | giấu trần | 3 kết quả, 2HP | ✅ |
| D3 | "resort nghỉ dưỡng 25m2..." (alt keyword) | giấu trần | 3 kết quả, 2.5HP | ✅ |

### Nhóm E — Tương tác với các tín hiệu khác

| # | Query | Kỳ vọng | Kết quả | Đánh giá |
|---|---|---|---|---|
| E1 | "biệt thự 20m2 nhưng tao muốn máy treo tường" | User override thắng building-type default | 13 kết quả, treo tường (không phải giấu trần) | ✅ Override đúng ưu tiên |
| E2 | "quán cafe Daikin 20m2 dưới 20 triệu" | Có thể 0 nếu giá thật cao hơn | 0 kết quả | ✅ Honest — verify DB: Daikin âm trần 2HP rẻ nhất là 26.2tr, thật sự không có SP nào <20tr |
| E3 | "nhà hàng 20m2 rẻ nhất" | Category=âm trần, 2HP, sort giá tăng dần | 3 kết quả, đúng sort | ✅ (đã sửa — bug #1) |
| E4 | "biệt thự nên lắp máy lạnh Daikin nào" (không có diện tích) | Category=giấu trần, không giới hạn HP, toàn bộ Daikin giấu trần | 21 kết quả | ✅ (đã sửa — bug #2) |

### Nhóm F — So sánh heat_load_factor cùng diện tích 30m²

| # | Loại công trình | heat_load_factor | Diện tích hiệu chỉnh | Tier | Kết quả |
|---|---|---|---|---|---|
| F1 | Văn phòng | 1.1x | 33m² | 2.5HP | 3 kết quả (âm trần) |
| F2 | Nhà xưởng | 1.5x | 45m² | 3HP | 7 kết quả (treo tường, không ép category) |
| F3 | Phòng server | 1.8x | 54m² | 4HP | 10 kết quả (âm trần — do 4HP vượt ngưỡng treo tường, tự mở rộng) |

Đúng theo thứ tự tăng dần công suất khi heat_load_factor tăng, với cùng 30m² đầu vào — xác nhận công thức nhân hệ số trước khi tra bảng hoạt động chính xác qua nhiều mức.

## Bug tìm thấy & đã sửa trong quá trình test

1. **Classifier đoán nhầm "khac" cho câu ngắn dạng "[công trình] [diện tích] [sort directive]"** (E3: "nhà hàng 20m2 rẻ nhất" ban đầu confidence 0.68 cho nhãn sai, rơi vào fallback path không có `CategorySlugs` bảo vệ, ra 21 kết quả không lọc tier). Nguyên nhân: câu ngắn, không giống các template training có sẵn. **Sửa:** bổ sung 4 câu ví dụ ngắn dạng này vào training data, retrain.

2. **Từ khóa loại công trình bị lọt vào leftover Search** — building-type category/heat-load đã hoạt động đúng qua `CategorySlugs`/`heat_load_factor`, nhưng chính từ khóa phát hiện được ("biệt thự") không bị loại khỏi phần text còn lại, nên bị gói thành cụm bắt buộc → chỉ match 2/21 sản phẩm tình cờ có chữ đó trong mô tả. **Sửa:** strip từ khóa building-type khỏi leftover, tương tự cách đã làm với canonical/brand/subcategory trước đó.

3. **Bug lồng trong bug #2**: fix ban đầu chỉ strip từ khóa ĐẦU TIÊN khớp, bỏ sót trường hợp message chứa 2 từ khóa chồng lấn của cùng 1 loại công trình (vd "nhà xưởng sản xuất" chứa cả "nhà xưởng" và "xưởng sản xuất", dùng chung chữ "xưởng"). Strip tuần tự theo cụm làm mất từ "xưởng" trước khi kịp kiểm tra cụm thứ 2, để sót "sản xuất" lọt thành cụm bắt buộc riêng → cắt từ 11 xuống còn 1 kết quả. **Sửa:** đổi từ xóa theo cụm-tuần-tự sang xóa theo **tập từ** (word-set) của toàn bộ từ khóa building-type đó — không còn phụ thuộc thứ tự xử lý.

## Kết luận

**26/26 query đúng sau khi sửa 3 bug** (đều thuộc cùng lớp lỗi: leftover text bị gói thành cụm tìm kiếm bắt buộc — pattern lỗi tái diễn nhiều lần trong suốt quá trình phát triển tính năng này, mỗi lần một biến thể mới). Verify được: mapping category chính xác cho 7 loại công trình có hàng thật, honest fallback (không hallucinate category) cho 6 loại công trình catalog không đáp ứng, heat_load_factor áp dụng đúng theo thứ tự tăng dần qua nhiều mức test, và user override luôn thắng building-type inference.

Tính năng sẵn sàng release. Điểm cần lưu ý dài hạn: mọi lần thêm signal mới (brand, subcategory, building type...) đều có nguy cơ tái diễn "leftover text bị gói nhầm thành cụm bắt buộc" — nên coi đây là checklist bắt buộc khi thêm bất kỳ loại detect mới nào trong tương lai, không chỉ riêng building taxonomy.
