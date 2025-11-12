-- Drop orders table
DROP TABLE IF EXISTS orders;

-- Drop trigger function if it's not used by other tables
-- Note: We won't drop the function if other tables are using it
-- DROP FUNCTION IF EXISTS update_updated_at_column();
