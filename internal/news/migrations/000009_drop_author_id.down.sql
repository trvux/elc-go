-- Plain uuid: the authors table it used to reference is gone, so the
-- foreign key cannot be restored.
ALTER TABLE news ADD COLUMN IF NOT EXISTS author_id uuid;
