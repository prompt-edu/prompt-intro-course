BEGIN;

-- Dropping the table also drops idx_peer_assignment_peer.
DROP TABLE IF EXISTS peer_assignment;

COMMIT;
