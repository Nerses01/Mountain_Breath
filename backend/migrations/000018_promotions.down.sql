-- Reverse of 000018: drop the discount split from orders, then the promo
-- tables in dependency order (children before parents).

ALTER TABLE orders
    DROP CONSTRAINT orders_discount_split;

ALTER TABLE orders
    DROP COLUMN member_discount_minor,
    DROP COLUMN promo_discount_minor,
    DROP COLUMN promo_code;

DROP TABLE cart_promos;
DROP TABLE promo_redemptions;
DROP TABLE promo_code_values;
DROP TABLE promo_codes;
