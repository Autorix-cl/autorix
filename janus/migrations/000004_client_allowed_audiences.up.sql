ALTER TABLE oauth2_clients
    ADD COLUMN allowed_audiences TEXT[] NOT NULL DEFAULT '{}';
