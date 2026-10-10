-- MealSwipe initial schema (PostgreSQL 15+)
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext NOT NULL UNIQUE,
    password_hash text   NOT NULL,
    display_name  text   NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,                -- SHA-256 of the opaque cookie token
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

CREATE TABLE ingredients (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          text NOT NULL UNIQUE,           -- canonical lowercase name
    category      text NOT NULL DEFAULT 'other'
                  CHECK (category IN ('produce','dairy','meat','fish','bakery','grains','frozen','canned','spices','oils','other')),
    unit_family   text NOT NULL DEFAULT 'count'
                  CHECK (unit_family IN ('mass','volume','count')),
    allergen_tags text[] NOT NULL DEFAULT '{}'
);

CREATE TABLE recipes (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                text NOT NULL UNIQUE,
    title               text NOT NULL,
    description         text NOT NULL DEFAULT '',
    instructions        text[] NOT NULL DEFAULT '{}',
    prep_minutes        int  NOT NULL DEFAULT 0 CHECK (prep_minutes >= 0),
    cook_minutes        int  NOT NULL DEFAULT 0 CHECK (cook_minutes >= 0),
    servings            numeric(6,2) NOT NULL DEFAULT 2 CHECK (servings > 0),
    meal_types          text[] NOT NULL DEFAULT '{}',
    category            text NOT NULL DEFAULT '',
    tags                text[] NOT NULL DEFAULT '{}',
    diet_flags          text[] NOT NULL DEFAULT '{}',
    image_url           text,
    image_credit        text,
    cost_per_serving    numeric(10,2) CHECK (cost_per_serving IS NULL OR cost_per_serving >= 0),
    kcal_per_serving    int CHECK (kcal_per_serving IS NULL OR kcal_per_serving >= 0),
    nutrition_estimate  boolean NOT NULL DEFAULT true,
    source              text NOT NULL DEFAULT 'seed',
    external_id         text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id)
);
CREATE INDEX recipes_meal_types_idx ON recipes USING gin (meal_types);

CREATE TABLE recipe_ingredients (
    recipe_id     uuid   NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    ingredient_id bigint NOT NULL REFERENCES ingredients(id),
    quantity      numeric(10,3) NOT NULL DEFAULT 0 CHECK (quantity >= 0),  -- 0 = "to taste"
    unit          text NOT NULL DEFAULT 'pcs',
    note          text NOT NULL DEFAULT '',
    position      int  NOT NULL DEFAULT 0,
    PRIMARY KEY (recipe_id, ingredient_id)
);
CREATE INDEX recipe_ingredients_ingredient_idx ON recipe_ingredients (ingredient_id);

CREATE TABLE user_preferences (
    user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    household_size   int NOT NULL DEFAULT 2 CHECK (household_size BETWEEN 1 AND 20),
    meals_per_day    int NOT NULL DEFAULT 3 CHECK (meals_per_day BETWEEN 1 AND 6),
    meal_types       text[] NOT NULL DEFAULT '{breakfast,lunch,dinner}',
    weekly_budget    numeric(10,2) CHECK (weekly_budget IS NULL OR weekly_budget >= 0),
    max_cook_minutes int CHECK (max_cook_minutes IS NULL OR max_cook_minutes > 0),
    diet_type        text NOT NULL DEFAULT 'any'
                     CHECK (diet_type IN ('any','vegetarian','vegan','pescatarian')),
    allergens        text[] NOT NULL DEFAULT '{}',
    goal             text NOT NULL DEFAULT '',
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_excluded_ingredients (
    user_id       uuid   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ingredient_id bigint NOT NULL REFERENCES ingredients(id),
    PRIMARY KEY (user_id, ingredient_id)
);

CREATE TABLE swipes (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id  uuid NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    direction  text NOT NULL CHECK (direction IN ('like','dislike')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, recipe_id)
);
CREATE INDEX swipes_user_direction_idx ON swipes (user_id, direction);

CREATE TABLE favorites (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipe_id  uuid NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, recipe_id)
);

CREATE TABLE menus (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    week_start date NOT NULL,
    status     text NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, week_start)
);

CREATE TABLE menu_items (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_id   uuid NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    day_index int  NOT NULL CHECK (day_index BETWEEN 0 AND 6),
    meal_type text NOT NULL,
    recipe_id uuid NOT NULL REFERENCES recipes(id),
    servings  numeric(6,2) NOT NULL DEFAULT 2 CHECK (servings > 0),
    locked    boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (menu_id, day_index, meal_type)
);
CREATE INDEX menu_items_menu_idx ON menu_items (menu_id);

CREATE TABLE shopping_lists (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    menu_id    uuid REFERENCES menus(id) ON DELETE SET NULL,
    name       text NOT NULL DEFAULT 'Список покупок',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, menu_id)            -- one list per menu (menu_id NULL = manual lists)
);

CREATE TABLE shopping_list_items (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id       uuid NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
    ingredient_id bigint REFERENCES ingredients(id),
    name          text NOT NULL,
    quantity      numeric(10,3) CHECK (quantity IS NULL OR quantity >= 0),
    unit          text NOT NULL DEFAULT '',
    category      text NOT NULL DEFAULT 'other',
    checked       boolean NOT NULL DEFAULT false,
    is_manual     boolean NOT NULL DEFAULT false,
    position      int NOT NULL DEFAULT 0
);
CREATE INDEX shopping_list_items_list_idx ON shopping_list_items (list_id);

CREATE TABLE pantry_items (
    user_id       uuid   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ingredient_id bigint NOT NULL REFERENCES ingredients(id),
    PRIMARY KEY (user_id, ingredient_id)
);
