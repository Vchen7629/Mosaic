CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS profiles (
    id SERIAL PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS profile_face_embeddings (
    id SERIAL PRIMARY KEY,
    profile_id INTEGER REFERENCES profiles(id),
    face_embedding vector(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS visitor_face_embeddings (
    id SERIAL PRIMARY KEY,
    profile_id INTEGER REFERENCES profiles(id),
    visitor_name VARCHAR NOT NULL,
    face_embedding vector(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_records (
    id SERIAL PRIMARY KEY,
    profile_id INTEGER REFERENCES profiles(id),
    visitor_id INTEGER REFERENCES visitor_face_embeddings(id),
    created_at TIMESTAMP DEFAULT NOW(),
    convo_text VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS briefings (
    id SERIAL PRIMARY KEY,
    profile_id INTEGER REFERENCES profiles(id),
    visitor_id INTEGER REFERENCES visitor_face_embeddings(id),
    briefing_text VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    profile_id    INT PRIMARY KEY REFERENCES profiles(id),
    session_token TEXT NOT NULL UNIQUE
);

CREATE UNIQUE INDEX ON visitor_face_embeddings(profile_id, visitor_name);
CREATE UNIQUE INDEX ON briefings(profile_id, visitor_id);
CREATE INDEX idx_sessions_session_token ON sessions(session_token);
CREATE INDEX idx_visitor_face_embedding ON visitor_face_embeddings USING ivfflat (face_embedding vector_l2_ops);
CREATE INDEX idx_patient_face_embedding ON profile_face_embeddings USING ivfflat (face_embedding vector_l2_ops);
