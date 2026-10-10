BEGIN;

-- Names of the phase's participants, cached from core so the tutor teams can be
-- served to callers (e.g. students) that may not read core's participation list.
CREATE TABLE participant_name (
  course_phase_id uuid NOT NULL,
  course_participation_id uuid NOT NULL,
  first_name text NOT NULL,
  last_name text NOT NULL,
  PRIMARY KEY (course_phase_id, course_participation_id)
);

COMMIT;
