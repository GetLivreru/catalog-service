CREATE TABLE product (
    id BIGSERIAL NOT NULL,
    guid UUID NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price BIGINT NOT NULL,
    category_guid UUID NOT NULL REFERENCES category(guid) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
