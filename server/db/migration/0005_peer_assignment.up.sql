BEGIN;

CREATE TABLE peer_assignment (
  course_phase_id uuid NOT NULL,
  student_id uuid NOT NULL,
  peer_id uuid NOT NULL,
  PRIMARY KEY (course_phase_id, student_id, peer_id),
  CHECK (student_id <> peer_id)
);

CREATE INDEX idx_peer_assignment_peer ON peer_assignment (course_phase_id, peer_id);

COMMIT;
