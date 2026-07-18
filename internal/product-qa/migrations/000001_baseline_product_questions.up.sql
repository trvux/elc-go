CREATE TABLE IF NOT EXISTS product_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    asker_name TEXT NOT NULL,
    asker_email TEXT,
    question_text TEXT NOT NULL,
    answer_text TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'rejected')),
    is_published BOOLEAN NOT NULL DEFAULT false,
    answered_at TIMESTAMPTZ,
    source_ip TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_product_questions_product_id ON product_questions (product_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_product_questions_status ON product_questions (status) WHERE deleted_at IS NULL;
