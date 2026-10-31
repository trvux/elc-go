-- The site settings feature (/settings API + admin page) was removed; its two
-- values (company_name, company_short_desc) are now constants in the frontend.
-- Applied in the same deploy that removes the route: the frontend stopped
-- calling /settings before this shipped, so nothing reads the table.
DROP TABLE IF EXISTS site_settings;
