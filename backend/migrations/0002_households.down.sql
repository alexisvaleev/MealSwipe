-- Reverts 0002: household-scoped rows fold back to their earliest member;
-- username becomes email again. Multi-member data cannot be un-merged.

ALTER TABLE pantry_items DROP CONSTRAINT pantry_items_pkey;
ALTER TABLE pantry_items DROP CONSTRAINT pantry_items_household_id_fkey;
UPDATE pantry_items p SET household_id =
    (SELECT u.id FROM users u WHERE u.household_id = p.household_id
     ORDER BY u.created_at LIMIT 1);
ALTER TABLE pantry_items RENAME COLUMN household_id TO user_id;
ALTER TABLE pantry_items ADD CONSTRAINT pantry_items_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE pantry_items ADD PRIMARY KEY (user_id, ingredient_id);

ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_household_id_menu_id_key;
ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_household_id_fkey;
UPDATE shopping_lists l SET household_id =
    (SELECT u.id FROM users u WHERE u.household_id = l.household_id
     ORDER BY u.created_at LIMIT 1);
ALTER TABLE shopping_lists RENAME COLUMN household_id TO user_id;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_user_id_menu_id_key
    UNIQUE (user_id, menu_id);

ALTER TABLE menus DROP CONSTRAINT menus_household_id_week_start_key;
ALTER TABLE menus DROP CONSTRAINT menus_household_id_fkey;
UPDATE menus m SET household_id =
    (SELECT u.id FROM users u WHERE u.household_id = m.household_id
     ORDER BY u.created_at LIMIT 1);
ALTER TABLE menus RENAME COLUMN household_id TO user_id;
ALTER TABLE menus ADD CONSTRAINT menus_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE menus ADD CONSTRAINT menus_user_id_week_start_key
    UNIQUE (user_id, week_start);

ALTER TABLE favorites DROP COLUMN added_by;
ALTER TABLE favorites DROP CONSTRAINT favorites_pkey;
ALTER TABLE favorites DROP CONSTRAINT favorites_household_id_fkey;
UPDATE favorites f SET household_id =
    (SELECT u.id FROM users u WHERE u.household_id = f.household_id
     ORDER BY u.created_at LIMIT 1);
ALTER TABLE favorites RENAME COLUMN household_id TO user_id;
ALTER TABLE favorites ADD CONSTRAINT favorites_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE favorites ADD PRIMARY KEY (user_id, recipe_id);

ALTER TABLE users DROP COLUMN household_joined_at;
ALTER TABLE users DROP COLUMN household_id;
DROP TABLE households;

ALTER TABLE users RENAME COLUMN username TO email;
ALTER INDEX users_username_key RENAME TO users_email_key;
