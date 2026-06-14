-- Hotel Key PMS initial schema
-- Run with: psql $DATABASE_URL -f 001_initial_schema.sql

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Tenants (one row per hotel chain or independent hotel)
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT        NOT NULL,
    code       TEXT        NOT NULL UNIQUE,
    plan       TEXT        NOT NULL DEFAULT 'starter',
    active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Hotels
CREATE TABLE hotels (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id),
    name        TEXT        NOT NULL,
    code        TEXT        NOT NULL,
    country     CHAR(2)     NOT NULL,
    city        TEXT        NOT NULL,
    address     TEXT        NOT NULL,
    phone       TEXT        NOT NULL,
    email       TEXT        NOT NULL,
    star_rating SMALLINT    NOT NULL DEFAULT 3 CHECK (star_rating BETWEEN 1 AND 5),
    active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE INDEX idx_hotels_tenant_id ON hotels (tenant_id);

-- Room types
CREATE TABLE room_types (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hotel_id        UUID        NOT NULL REFERENCES hotels(id),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id),
    name            TEXT        NOT NULL,
    code            TEXT        NOT NULL,
    max_occupancy   SMALLINT    NOT NULL DEFAULT 2,
    base_rate       NUMERIC(12,2) NOT NULL,
    currency        CHAR(3)     NOT NULL DEFAULT 'JPY',
    -- Per-person pricing hook (Japan/China markets)
    per_person_pricing_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hotel_id, code)
);

CREATE INDEX idx_room_types_hotel_id ON room_types (hotel_id);
CREATE INDEX idx_room_types_tenant_id ON room_types (tenant_id);

-- Rooms
CREATE TABLE rooms (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hotel_id     UUID        NOT NULL REFERENCES hotels(id),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id),
    room_type_id UUID        NOT NULL REFERENCES room_types(id),
    number       TEXT        NOT NULL,
    floor        SMALLINT,
    status       TEXT        NOT NULL DEFAULT 'clean'
                             CHECK (status IN ('clean','dirty','inspected','out_of_order','out_of_service')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hotel_id, number)
);

CREATE INDEX idx_rooms_hotel_id   ON rooms (hotel_id);
CREATE INDEX idx_rooms_tenant_id  ON rooms (tenant_id);
CREATE INDEX idx_rooms_status     ON rooms (hotel_id, status);

-- Guests
CREATE TABLE guests (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id),
    first_name  TEXT        NOT NULL,
    last_name   TEXT        NOT NULL,
    email       TEXT,
    phone       TEXT,
    nationality CHAR(2),
    passport_no TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_guests_tenant_id ON guests (tenant_id);
CREATE INDEX idx_guests_email     ON guests (tenant_id, email) WHERE email IS NOT NULL;

-- Reservations
CREATE TABLE reservations (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id),
    hotel_id      UUID        NOT NULL REFERENCES hotels(id),
    room_id       UUID        REFERENCES rooms(id),
    room_type_id  UUID        NOT NULL REFERENCES room_types(id),
    guest_id      UUID        NOT NULL REFERENCES guests(id),
    status        TEXT        NOT NULL DEFAULT 'confirmed'
                              CHECK (status IN ('tentative','confirmed','checked_in','checked_out','cancelled','no_show')),
    arrival_date  DATE        NOT NULL,
    departure_date DATE       NOT NULL,
    num_adults    SMALLINT    NOT NULL DEFAULT 1,
    num_children  SMALLINT    NOT NULL DEFAULT 0,
    rate_plan     TEXT        NOT NULL DEFAULT 'BAR',
    total_amount  NUMERIC(14,2),
    currency      CHAR(3)     NOT NULL DEFAULT 'JPY',
    channel       TEXT        NOT NULL DEFAULT 'direct',
    external_ref  TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (departure_date > arrival_date)
);

CREATE INDEX idx_reservations_tenant_id    ON reservations (tenant_id);
CREATE INDEX idx_reservations_hotel_id     ON reservations (hotel_id);
CREATE INDEX idx_reservations_guest_id     ON reservations (guest_id);
CREATE INDEX idx_reservations_dates        ON reservations (hotel_id, arrival_date, departure_date);
CREATE INDEX idx_reservations_status       ON reservations (hotel_id, status);

-- Folio (billing ledger — append-only enforced by trigger)
CREATE TABLE folio_entries (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      UUID           NOT NULL REFERENCES tenants(id),
    reservation_id UUID           NOT NULL REFERENCES reservations(id),
    entry_type     TEXT           NOT NULL
                                  CHECK (entry_type IN ('charge','payment','tax','discount','refund','void')),
    description    TEXT           NOT NULL,
    amount         NUMERIC(14,2)  NOT NULL,
    currency       CHAR(3)        NOT NULL DEFAULT 'JPY',
    posted_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    posted_by      TEXT           NOT NULL
);

CREATE INDEX idx_folio_entries_reservation_id ON folio_entries (reservation_id);
CREATE INDEX idx_folio_entries_tenant_id      ON folio_entries (tenant_id);

-- Trigger to make folio_entries truly append-only
CREATE OR REPLACE FUNCTION prevent_folio_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'folio_entries is append-only; issue a void/refund entry instead';
END;
$$;

CREATE TRIGGER folio_no_update
    BEFORE UPDATE ON folio_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_folio_mutation();

CREATE TRIGGER folio_no_delete
    BEFORE DELETE ON folio_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_folio_mutation();

-- Housekeeping tasks
CREATE TABLE housekeeping_tasks (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id),
    hotel_id    UUID        NOT NULL REFERENCES hotels(id),
    room_id     UUID        NOT NULL REFERENCES rooms(id),
    task_type   TEXT        NOT NULL CHECK (task_type IN ('checkout','stayover','turndown','deep_clean','inspection')),
    status      TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','done','skipped')),
    assigned_to TEXT,
    scheduled_for DATE      NOT NULL,
    completed_at TIMESTAMPTZ,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hk_tasks_hotel_date ON housekeeping_tasks (hotel_id, scheduled_for, status);

-- Row Level Security
ALTER TABLE hotels            ENABLE ROW LEVEL SECURITY;
ALTER TABLE room_types        ENABLE ROW LEVEL SECURITY;
ALTER TABLE rooms             ENABLE ROW LEVEL SECURITY;
ALTER TABLE guests            ENABLE ROW LEVEL SECURITY;
ALTER TABLE reservations      ENABLE ROW LEVEL SECURITY;
ALTER TABLE folio_entries     ENABLE ROW LEVEL SECURITY;
ALTER TABLE housekeeping_tasks ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON hotels
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON room_types
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON rooms
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON guests
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON reservations
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON folio_entries
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

CREATE POLICY tenant_isolation ON housekeeping_tasks
    USING (tenant_id::TEXT = current_setting('app.current_tenant_id', TRUE));

-- updated_at auto-maintenance
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TRIGGER set_updated_at BEFORE UPDATE ON hotels
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON room_types
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON rooms
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON guests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON reservations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON housekeeping_tasks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
