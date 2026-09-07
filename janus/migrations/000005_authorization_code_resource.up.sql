ALTER TABLE oauth2_login_challenges ADD COLUMN resource TEXT;
ALTER TABLE oauth2_consent_challenges ADD COLUMN resource TEXT;
ALTER TABLE oauth2_grants ADD COLUMN resource TEXT;
