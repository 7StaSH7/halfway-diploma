-- Rename order_number column back to order in withdrawals table
ALTER TABLE withdrawals RENAME COLUMN order_number TO "order";

-- Update the index name to reflect the column name change
DROP INDEX IF EXISTS idx_withdrawals_order_number;
CREATE INDEX idx_withdrawals_order ON withdrawals("order");
