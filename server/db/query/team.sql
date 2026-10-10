-- A student counts for the first seat by name that has a tutor and is not a tutor seat.
-- GetTutorTeams, GetTeamAllocations and GetTeamAllocation all apply this rule, so a
-- student seated twice still belongs to exactly one team.

-- One team per tutor: the tutor plus every student seated in one of their seats.
-- name: GetTutorTeams :many
WITH student_seat AS (
    SELECT DISTINCT ON (s.assigned_student) s.assigned_student, s.assigned_tutor, s.seat_name
    FROM seat s
    WHERE s.course_phase_id = $1
      AND s.assigned_student IS NOT NULL
      AND s.assigned_tutor IS NOT NULL
      AND NOT s.is_tutor_seat
    ORDER BY s.assigned_student, s.seat_name
)
SELECT t.id,
       t.first_name,
       t.last_name,
       COALESCE(members.team_members, '[]'::jsonb)::jsonb AS team_members
FROM tutor t
         LEFT JOIN LATERAL (
    SELECT jsonb_agg(
                   jsonb_build_object(
                           'id', ss.assigned_student,
                           'firstName', COALESCE(pn.first_name, ''),
                           'lastName', COALESCE(pn.last_name, '')
                   )
                   ORDER BY pn.first_name, pn.last_name, ss.seat_name
           ) AS team_members
    FROM student_seat ss
             LEFT JOIN participant_name pn
                       ON pn.course_phase_id = t.course_phase_id
                           AND pn.course_participation_id = ss.assigned_student
    WHERE ss.assigned_tutor = t.id
    ) members ON TRUE
WHERE t.course_phase_id = $1
ORDER BY t.first_name, t.last_name, t.id;

-- name: GetTeamAllocations :many
SELECT DISTINCT ON (s.assigned_student) s.assigned_student::uuid AS course_participation_id,
                                        s.assigned_tutor::uuid   AS team_id
FROM seat s
WHERE s.course_phase_id = $1
  AND s.assigned_student IS NOT NULL
  AND s.assigned_tutor IS NOT NULL
  AND NOT s.is_tutor_seat
ORDER BY s.assigned_student, s.seat_name;

-- name: GetTeamAllocation :one
SELECT s.assigned_tutor::uuid AS team_id
FROM seat s
WHERE s.course_phase_id = $1
  AND s.assigned_student = sqlc.arg(course_participation_id)::uuid
  AND s.assigned_tutor IS NOT NULL
  AND NOT s.is_tutor_seat
ORDER BY s.seat_name
LIMIT 1;

-- name: GetTutorTeamByUniversityLogin :one
SELECT id
FROM tutor
WHERE course_phase_id = $1
  AND lower(trim(university_login)) = sqlc.arg(university_login)::text
ORDER BY id
LIMIT 1;

-- The arrays are zipped by position (set-returning functions in the select list
-- advance in lockstep), so callers must pass them with equal lengths. Unchanged
-- names are left alone to keep repeated refreshes from rewriting every row.
-- name: UpsertParticipantNames :exec
INSERT INTO participant_name (course_phase_id, course_participation_id, first_name, last_name)
SELECT sqlc.arg(course_phase_id)::uuid,
       unnest(sqlc.arg(course_participation_ids)::uuid[]),
       unnest(sqlc.arg(first_names)::text[]),
       unnest(sqlc.arg(last_names)::text[])
ON CONFLICT (course_phase_id, course_participation_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name  = EXCLUDED.last_name
WHERE participant_name.first_name IS DISTINCT FROM EXCLUDED.first_name
   OR participant_name.last_name IS DISTINCT FROM EXCLUDED.last_name;
