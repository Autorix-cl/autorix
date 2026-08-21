CREATE TABLE oauth2_login_challenges (
    challenge VARCHAR(128) PRIMARY KEY,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    response_type TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    state TEXT,
    nonce TEXT,
    code_challenge TEXT,
    code_challenge_method TEXT,
    subject VARCHAR(255),
    login_verifier VARCHAR(128),
    handled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE oauth2_consent_challenges (
    challenge VARCHAR(128) PRIMARY KEY,
    login_challenge VARCHAR(128) NOT NULL REFERENCES oauth2_login_challenges(challenge) ON DELETE CASCADE,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth2_clients(id) ON DELETE CASCADE,
    subject VARCHAR(255) NOT NULL,
    requested_scopes TEXT[] NOT NULL,
    granted_scopes TEXT[],
    consent_verifier VARCHAR(128),
    handled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
