-- Create index on orders status for faster queries
CREATE INDEX idx_orders_status ON orders(status);
