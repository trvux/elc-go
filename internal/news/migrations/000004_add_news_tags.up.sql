CREATE TABLE IF NOT EXISTS news_tags (
    news_id     uuid NOT NULL REFERENCES news(id) ON DELETE CASCADE,
    tag_id      uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (news_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_news_tags_tag_id ON news_tags (tag_id);
