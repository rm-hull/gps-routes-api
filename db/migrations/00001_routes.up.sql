CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE routes (
    object_id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    ref TEXT NOT NULL,
    title TEXT NOT NULL,
    headline_image_url TEXT,
    gpx_url TEXT,
    _geoloc GEOMETRY(Point, 4326) NOT NULL,
    distance_km NUMERIC(10, 2),
    description TEXT NOT NULL,
    video_url TEXT,
    display_address TEXT,
    postcode TEXT,
    district TEXT,
    county TEXT,
    region TEXT,
    state TEXT,
    country TEXT,
    estimated_duration TEXT,
    difficulty TEXT,
    terrain TEXT[],
    points_of_interest TEXT[],
    facilities TEXT[],
    route_type TEXT,
    activities TEXT[],
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'C') ||
        setweight(to_tsvector('english', coalesce(display_address, '')), 'B')
    ) STORED
);

CREATE INDEX idx_routes_district ON routes (district);
CREATE INDEX idx_routes_county ON routes (county);
CREATE INDEX idx_routes_region ON routes (region);
CREATE INDEX idx_routes_state ON routes (state);
CREATE INDEX idx_routes_country ON routes (country);
CREATE INDEX idx_routes_route_type ON routes (route_type);
CREATE INDEX idx_routes_estimated_duration ON routes (estimated_duration);
CREATE INDEX idx_routes_difficulty ON routes (difficulty);
CREATE INDEX idx_routes_terrain ON routes USING GIN (terrain);
CREATE INDEX idx_routes_facilities ON routes USING GIN (facilities);
CREATE INDEX idx_routes_points_of_interest ON routes USING GIN (points_of_interest);
CREATE INDEX idx_routes_activities ON routes USING GIN (activities);
CREATE INDEX idx_routes_geolocation ON routes USING GIST (_geoloc);
CREATE INDEX idx_routes_search_vector ON routes USING GIN (search_vector);

CREATE TABLE nearby (
    id SERIAL PRIMARY KEY,
    route_object_id TEXT REFERENCES routes(object_id) ON DELETE CASCADE,
    description TEXT,
    object_id TEXT,
    ref TEXT
);

CREATE TABLE images (
    id SERIAL PRIMARY KEY,
    route_object_id TEXT REFERENCES routes(object_id) ON DELETE CASCADE,
    src TEXT NOT NULL,
    title TEXT,
    caption TEXT
);

CREATE TABLE details (
    id SERIAL PRIMARY KEY,
    route_object_id TEXT REFERENCES routes(object_id) ON DELETE CASCADE,
    subtitle TEXT NOT NULL,
    content TEXT NOT NULL
);
