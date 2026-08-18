#!/usr/bin/env python3
"""One-time bulk content generator for hp_pages.content (TipTap JSON).
Writes SQL UPDATE statements to stdout; run via psql -f.
Not idempotent-guarded on purpose — content is admin-editable afterward,
this is just the initial SEO copy seed.
"""
import json

# HP -> (BTU, room area, current draw, machine type, phase)
SPECS = [
    ("1",   "may-lanh-1hp",  "9.000",           "9 – 15",   "4 – 5",   "treo tường",              "1 pha"),
    ("1.5", "may-lanh-15hp", "12.000",          "15 – 20",  "6 – 7",   "treo tường",              "1 pha"),
    ("2",   "may-lanh-2hp",  "18.000",          "20 – 30",  "8 – 9",   "treo tường hoặc áp trần", "1 pha"),
    ("2.5", "may-lanh-25hp", "21.000 – 24.000", "30 – 40",  "10 – 11", "áp trần hoặc tủ đứng",    "1 pha"),
    ("3",   "may-lanh-3hp",  "28.000",          "40 – 50",  "13 – 14", "áp trần hoặc tủ đứng",    "1 pha hoặc 3 pha"),
    ("3.5", "may-lanh-35hp", "36.000",          "50 – 60",  "15 – 16", "áp trần hoặc tủ đứng",    "1 pha hoặc 3 pha"),
    ("4",   "may-lanh-4hp",  "40.000 – 48.000", "60 – 80",  "18 – 20", "áp trần hoặc tủ đứng",    "3 pha phổ biến"),
    ("4.5", "may-lanh-45hp", "48.000 – 52.000", "80 – 90",  "20 – 22", "áp trần hoặc tủ đứng",    "3 pha"),
    ("5",   "may-lanh-5hp",  "60.000",          "90 – 100", "24 – 26", "áp trần hoặc tủ đứng",    "3 pha"),
    ("5.5", "may-lanh-55hp", "62.000 – 65.000", "100 – 120","27 – 28", "áp trần hoặc tủ đứng",    "3 pha"),
    ("6",   "may-lanh-6hp",  "68.000 – 70.000", "120 – 150","30 – 32", "áp trần, tủ đứng hoặc multi", "3 pha"),
    ("10",  "may-lanh-10hp", "96.000 – 120.000","200 – 250","45 – 50", "áp trần công suất lớn hoặc multi", "3 pha"),
]

def hp_label(hp):
    return hp.replace(".", ",")  # "2.5" -> "2,5" for Vietnamese-style display text

def doc_for(hp, slug, btu, area, current, machine_type, phase):
    hp_disp = hp_label(hp)
    hp_compact = hp.replace(".", "")  # "2.5" -> "25" matches slug convention
    heading = f"Máy lạnh {hp}HP – Thông số kỹ thuật và diện tích phòng phù hợp"
    p1 = (
        f"Máy lạnh {hp}HP (còn gọi là máy lạnh {hp_disp} ngựa, công suất lạnh khoảng {btu} BTU/h) "
        f"là lựa chọn phổ biến cho không gian rộng {area}m². \"HP\" (Horse Power) và \"ngựa\" là hai "
        f"cách gọi cùng một đơn vị công suất máy lạnh, thường được người dùng tìm kiếm thay thế cho nhau "
        f"khi chọn mua điều hòa."
    )
    p2 = (
        f"Trước khi chọn máy lạnh {hp}HP, nên đối chiếu diện tích phòng thực tế, hướng nắng, số lượng "
        f"người sử dụng và mật độ thiết bị tỏa nhiệt trong phòng để chọn đúng công suất — máy quá nhỏ "
        f"so với phòng sẽ chạy liên tục, hao điện và làm lạnh chậm; máy quá lớn gây lãng phí chi phí đầu tư."
    )
    specs = [
        f"Công suất lạnh: khoảng {btu} BTU/h",
        f"Diện tích phòng phù hợp: {area} m²",
        f"Dòng điện tiêu thụ ước tính: {current} A",
        f"Loại máy phổ biến ở công suất này: {machine_type}",
        f"Nguồn điện: {phase}",
    ]
    p3 = (
        f"Điện máy ELC phân phối máy lạnh {hp}HP chính hãng từ nhiều thương hiệu — Daikin, LG, "
        f"Panasonic, Casper — với đầy đủ mức giá, kèm lắp đặt và bảo hành chính hãng. Các thông số "
        f"trên chỉ mang tính tham khảo, thông số kỹ thuật chính xác của từng model cụ thể được ghi "
        f"rõ trong bảng thông số sản phẩm bên dưới."
    )

    return {
        "type": "doc",
        "content": [
            {"type": "heading", "attrs": {"level": 2}, "content": [{"type": "text", "text": heading}]},
            {"type": "paragraph", "content": [{"type": "text", "text": p1}]},
            {"type": "paragraph", "content": [{"type": "text", "text": p2}]},
            {"type": "heading", "attrs": {"level": 3}, "content": [{"type": "text", "text": f"Thông số kỹ thuật máy lạnh {hp}HP"}]},
            {
                "type": "bulletList",
                "content": [
                    {"type": "listItem", "content": [{"type": "paragraph", "content": [{"type": "text", "text": s}]}]}
                    for s in specs
                ],
            },
            {"type": "paragraph", "content": [{"type": "text", "text": p3}]},
        ],
    }

def sql_escape(s: str) -> str:
    return s.replace("'", "''")

for hp, slug, btu, area, current, machine_type, phase in SPECS:
    content = doc_for(hp, slug, btu, area, current, machine_type, phase)
    content_json = sql_escape(json.dumps(content, ensure_ascii=False))
    print(f"UPDATE hp_pages SET content = '{content_json}'::jsonb WHERE slug = '{slug}';")
