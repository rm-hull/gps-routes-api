CREATE TABLE details (
    id SERIAL PRIMARY KEY,
    route_object_id TEXT REFERENCES routes(object_id) ON DELETE CASCADE,
    subtitle TEXT NOT NULL,
    content TEXT NOT NULL
);

