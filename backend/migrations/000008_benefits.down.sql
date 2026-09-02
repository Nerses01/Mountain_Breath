-- Reverse order of creation. product_benefits and benefit_translations both
-- reference benefits, so benefits goes last — DROP TABLE refuses while a
-- foreign key still points at it (the same protection that makes the
-- ON DELETE clauses above meaningful).
DROP TABLE IF EXISTS benefit_translations;
DROP TABLE IF EXISTS product_benefits;
DROP TABLE IF EXISTS benefits;
