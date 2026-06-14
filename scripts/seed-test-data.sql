-- Seed data for local development
-- Apply after 001_initial_schema.sql
-- Run: psql $DATABASE_URL -f seed-test-data.sql

BEGIN;

-- Tenants
INSERT INTO tenants (id, name, code, plan) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Sunrise Hotels Japan', 'sunrise-jp', 'enterprise'),
  ('00000000-0000-0000-0000-000000000002', 'Demo Hotel Group', 'demo', 'starter')
ON CONFLICT (code) DO NOTHING;

-- Hotels
INSERT INTO hotels (id, tenant_id, name, code, country, city, address, phone, email, star_rating) VALUES
  (
    '00000000-0000-0000-0001-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Sunrise Tokyo Shinjuku',
    'TSK',
    'JP',
    'Tokyo',
    '1-1 Shinjuku, Shinjuku-ku, Tokyo 160-0022',
    '+81-3-1234-5678',
    'tokyo@sunrise-hotels.jp',
    4
  ),
  (
    '00000000-0000-0000-0001-000000000002',
    '00000000-0000-0000-0000-000000000002',
    'Demo Hotel',
    'DEMO',
    'JP',
    'Osaka',
    '1-1 Namba, Chuo-ku, Osaka 542-0076',
    '+81-6-9876-5432',
    'demo@demo-hotel.jp',
    3
  )
ON CONFLICT DO NOTHING;

-- Room types (Tokyo)
INSERT INTO room_types (id, hotel_id, tenant_id, name, code, max_occupancy, base_rate, currency) VALUES
  ('00000000-0000-0001-0001-000000000001', '00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', 'Standard Twin', 'STD-TWN', 2, 18000, 'JPY'),
  ('00000000-0000-0001-0001-000000000002', '00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', 'Deluxe King', 'DLX-KNG', 2, 28000, 'JPY'),
  ('00000000-0000-0001-0001-000000000003', '00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', 'Suite', 'STE', 4, 55000, 'JPY')
ON CONFLICT DO NOTHING;

-- Rooms
INSERT INTO rooms (hotel_id, tenant_id, room_type_id, number, floor, status) VALUES
  ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0001-0001-000000000001', '101', 1, 'clean'),
  ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0001-0001-000000000001', '102', 1, 'clean'),
  ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0001-0001-000000000002', '201', 2, 'clean'),
  ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0001-0001-000000000003', '301', 3, 'clean')
ON CONFLICT DO NOTHING;

-- Sample guest
INSERT INTO guests (id, tenant_id, first_name, last_name, email, phone, nationality) VALUES
  ('00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000001', 'Taro', 'Yamada', 'taro.yamada@example.com', '+81-90-1234-5678', 'JP')
ON CONFLICT DO NOTHING;

-- Sample reservation
INSERT INTO reservations (tenant_id, hotel_id, room_type_id, guest_id, status, arrival_date, departure_date, num_adults, rate_plan, currency, channel) VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0001-000000000001',
    '00000000-0000-0001-0001-000000000002',
    '00000000-0000-0000-0002-000000000001',
    'confirmed',
    CURRENT_DATE + INTERVAL '1 day',
    CURRENT_DATE + INTERVAL '3 days',
    2,
    'BAR',
    'JPY',
    'direct'
  )
ON CONFLICT DO NOTHING;

COMMIT;
