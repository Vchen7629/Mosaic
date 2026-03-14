CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS patient (
    id SERIAL PRIMARY KEY,
    face_embedding vector(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS visitor_face_embeddings (
    id SERIAL PRIMARY KEY,
    patient_id INTEGER REFERENCES patient(id),
    visitor_name VARCHAR NOT NULL,
    face_embedding vector(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_records (
    id SERIAL PRIMARY KEY,
    patient_id INTEGER REFERENCES patient(id),
    visitor_id INTEGER REFERENCES visitor_face_embeddings(id),
    created_at TIMESTAMP DEFAULT NOW(),
    convo_text VARCHAR NOT NULL
);

CREATE TABLE IF NOT EXISTS briefings (
    id SERIAL PRIMARY KEY,
    patient_id INTEGER REFERENCES patient(id),
    visitor_id INTEGER REFERENCES visitor_face_embeddings(id),
    briefing_text VARCHAR NOT NULL
);

CREATE UNIQUE INDEX ON visitor_face_embeddings(patient_id, visitor_name);   
CREATE INDEX idx_visitor_face_embedding ON visitor_face_embeddings USING ivfflat (face_embedding vector_l2_ops);
CREATE INDEX idx_patient_face_embedding ON patient USING ivfflat (face_embedding vector_l2_ops);
