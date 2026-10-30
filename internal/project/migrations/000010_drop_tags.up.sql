-- The tags feature (tags + product_tags/news_tags/project_tags) was removed.
-- One statement so Postgres orders the drops itself (the three join tables
-- reference tags). Lives in project's migrations because it only needs to run
-- once; it is applied in a deploy AFTER the code stopped reading these tables,
-- so the previous binary never queried a missing table. schema_migrations_tag
-- was the tracking table of the deleted `tag` module.
DROP TABLE IF EXISTS product_tags, news_tags, project_tags, tags;
DROP TABLE IF EXISTS schema_migrations_tag;
