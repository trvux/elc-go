-- product_lines.code was being (ab)used to hold multiple MPN prefixes
-- concatenated with "-" (e.g. "FTF-FTC", "FTKY-FTKM-FTKZ") — works for
-- matching/display of a *single* string but can't be shown as separate
-- chips in the admin UI and isn't a real list. mpn_prefixes is the proper
-- array; `code` goes back to being a short, single, human-assigned
-- identifier for the line itself (unique per brand), not a concatenation.
ALTER TABLE product_lines ADD COLUMN IF NOT EXISTS mpn_prefixes TEXT[] NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_product_lines_mpn_prefixes ON product_lines USING gin (mpn_prefixes);
