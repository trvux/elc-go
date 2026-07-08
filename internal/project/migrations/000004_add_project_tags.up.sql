CREATE TABLE IF NOT EXISTS project_tags (
    project_id  uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag_id      uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_project_tags_tag_id ON project_tags (tag_id);
