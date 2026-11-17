BEGIN;

-- Lossy: Apple device UDIDs are usually not UUIDs (e.g. 00008030-001A35E11A88003A),
-- so only values that are valid UUIDs survive the rollback. Every other value
-- becomes NULL.

-- Rename apple_watch_udid back to apple_watch_uuid and restore the uuid type
ALTER TABLE developer_profile
  RENAME COLUMN apple_watch_udid TO apple_watch_uuid;
ALTER TABLE developer_profile
  ALTER COLUMN apple_watch_uuid TYPE uuid USING (
    CASE WHEN apple_watch_uuid ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
      THEN apple_watch_uuid::uuid
    END
  );

-- Rename ipad_udid back to ipad_uuid and restore the uuid type
ALTER TABLE developer_profile
  RENAME COLUMN ipad_udid TO ipad_uuid;
ALTER TABLE developer_profile
  ALTER COLUMN ipad_uuid TYPE uuid USING (
    CASE WHEN ipad_uuid ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
      THEN ipad_uuid::uuid
    END
  );

-- Rename iphone_udid back to iphone_uuid and restore the uuid type
ALTER TABLE developer_profile
  RENAME COLUMN iphone_udid TO iphone_uuid;
ALTER TABLE developer_profile
  ALTER COLUMN iphone_uuid TYPE uuid USING (
    CASE WHEN iphone_uuid ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
      THEN iphone_uuid::uuid
    END
  );

COMMIT;
