DROP TABLE IF EXISTS shipping_rates;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_totals_balance,
    DROP CONSTRAINT IF EXISTS orders_tax_contained;

ALTER TABLE orders
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS payment_status,
    DROP COLUMN IF EXISTS subtotal_minor,
    DROP COLUMN IF EXISTS shipping_minor,
    DROP COLUMN IF EXISTS discount_minor,
    DROP COLUMN IF EXISTS tax_minor,
    DROP COLUMN IF EXISTS ship_first_name,
    DROP COLUMN IF EXISTS ship_last_name,
    DROP COLUMN IF EXISTS ship_phone,
    DROP COLUMN IF EXISTS ship_street,
    DROP COLUMN IF EXISTS ship_city,
    DROP COLUMN IF EXISTS ship_postal_code,
    DROP COLUMN IF EXISTS ship_country,
    DROP COLUMN IF EXISTS delivery_note,
    DROP COLUMN IF EXISTS leave_with_neighbour;

-- Index goes with the table.
DROP TABLE IF EXISTS addresses;
