-- Reverse of 000019.

ALTER TABLE addresses
    DROP COLUMN label;

DROP TABLE oauth_identities;
DROP TABLE password_reset_tokens;
DROP TABLE wishlist_items;
