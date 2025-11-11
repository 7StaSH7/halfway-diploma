-- Rename order column to order_number in withdrawals table
ALTER TABLE withdrawals RENAME COLUMN "order" TO order_number;

-- Update the index name to reflect the column name change
DROP INDEX IF EXISTS idx_withdrawals_order;
CREATE INDEX idx_withdrawals_order_number ON withdrawals(order_number);
