CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    label TEXT,
    value TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    order_index INTEGER DEFAULT 0
);
