-- MealSwipe 0002: login-by-username + shared households.
-- Personal scope stays on user_id (preferences, swipes, sessions);
-- shared scope moves to household_id (favorites, menus, shopping lists, pantry).

CREATE TABLE households (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL DEFAULT 'Моя кухня',
    invite_code text NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- users: email -> username, membership + join timestamp
ALTER TABLE users RENAME COLUMN email TO username;
ALTER INDEX users_email_key RENAME TO users_username_key;
ALTER TABLE users ADD COLUMN household_id uuid REFERENCES households(id);
ALTER TABLE users ADD COLUMN household_joined_at timestamptz NOT NULL DEFAULT now();

-- One personal household per existing user keeps the 1:1 data mapping.
DO $$
DECLARE u RECORD; h uuid;
BEGIN
    FOR u IN SELECT id FROM users LOOP
        INSERT INTO households (invite_code)
            VALUES (upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 10)))
            RETURNING id INTO h;
        UPDATE users SET household_id = h WHERE id = u.id;
    END LOOP;
END $$;

ALTER TABLE users ALTER COLUMN household_id SET NOT NULL;
CREATE INDEX users_household_idx ON users (household_id);

-- favorites -> household scope (+ who bookmarked it — backfilled before the
-- user_id column is renamed away)
ALTER TABLE favorites ADD COLUMN added_by uuid REFERENCES users(id) ON DELETE SET NULL;
UPDATE favorites SET added_by = user_id;
ALTER TABLE favorites DROP CONSTRAINT favorites_user_id_fkey;
ALTER TABLE favorites DROP CONSTRAINT favorites_pkey;
ALTER TABLE favorites RENAME COLUMN user_id TO household_id;
UPDATE favorites f SET household_id = u.household_id FROM users u WHERE f.household_id = u.id;
ALTER TABLE favorites ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE favorites ADD CONSTRAINT favorites_household_id_fkey
    FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE;
ALTER TABLE favorites ADD PRIMARY KEY (household_id, recipe_id);

-- menus -> household scope
ALTER TABLE menus DROP CONSTRAINT menus_user_id_fkey;
ALTER TABLE menus DROP CONSTRAINT menus_user_id_week_start_key;
ALTER TABLE menus RENAME COLUMN user_id TO household_id;
UPDATE menus m SET household_id = u.household_id FROM users u WHERE m.household_id = u.id;
ALTER TABLE menus ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE menus ADD CONSTRAINT menus_household_id_fkey
    FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE;
ALTER TABLE menus ADD CONSTRAINT menus_household_id_week_start_key
    UNIQUE (household_id, week_start);

-- shopping_lists -> household scope
ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_user_id_fkey;
ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_user_id_menu_id_key;
ALTER TABLE shopping_lists RENAME COLUMN user_id TO household_id;
UPDATE shopping_lists l SET household_id = u.household_id FROM users u WHERE l.household_id = u.id;
ALTER TABLE shopping_lists ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_household_id_fkey
    FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_household_id_menu_id_key
    UNIQUE (household_id, menu_id);

-- pantry_items -> household scope
ALTER TABLE pantry_items DROP CONSTRAINT pantry_items_user_id_fkey;
ALTER TABLE pantry_items DROP CONSTRAINT pantry_items_pkey;
ALTER TABLE pantry_items RENAME COLUMN user_id TO household_id;
UPDATE pantry_items p SET household_id = u.household_id FROM users u WHERE p.household_id = u.id;
ALTER TABLE pantry_items ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE pantry_items ADD CONSTRAINT pantry_items_household_id_fkey
    FOREIGN KEY (household_id) REFERENCES households(id) ON DELETE CASCADE;
ALTER TABLE pantry_items ADD PRIMARY KEY (household_id, ingredient_id);
