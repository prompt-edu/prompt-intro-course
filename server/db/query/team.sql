-- One team per tutor: the tutor plus every student seated in one of their seats.
-- name: GetTutorTeams :many
SELECT t.id,
       t.first_name,
       t.last_name,
       COALESCE(members.team_members, '[]'::jsonb)::jsonb AS team_members
FROM tutor t
         LEFT JOIN LATERAL (
    SELECT jsonb_agg(
                   jsonb_build_object(
                           'id', s.assigned_student,
                           'firstName', COALESCE(pn.first_name, ''),
                           'lastName', COALESCE(pn.last_name, '')
                   )
                   ORDER BY pn.first_name, pn.last_name, s.seat_name
           ) AS team_members
    FROM seat s
             LEFT JOIN participant_name pn
                       ON pn.course_phase_id = s.course_phase_id
                           AND pn.course_participation_id = s.assigned_student
    WHERE s.course_phase_id = t.course_phase_id
      AND s.assigned_tutor = t.id
      AND s.assigned_student IS NOT NULL
    ) members ON TRUE
WHERE t.course_phase_id = $1
ORDER BY t.first_name, t.last_name;

-- A student seated twice counts for the first seat by name, matching GetTeamAllocation.
-- name: GetTeamAllocations :many
SELECT DISTINCT ON (s.assigned_student) s.assigned_student::uuid AS course_participation_id,
                                        t.id                     AS team_id
FROM seat s
         JOIN tutor t
              ON t.course_phase_id = s.course_phase_id
                  AND t.id = s.assigned_tutor
WHERE s.course_phase_id = $1
  AND s.assigned_student IS NOT NULL
ORDER BY s.assigned_student, s.seat_name;

-- name: GetTeamAllocation :one
SELECT t.id AS team_id
FROM seat s
         JOIN tutor t
              ON t.course_phase_id = s.course_phase_id
                  AND t.id = s.assigned_tutor
WHERE s.course_phase_id = $1
  AND s.assigned_student = sqlc.arg(course_participation_id)::uuid
ORDER BY s.seat_name
LIMIT 1;

-- The arrays are zipped by position (set-returning functions in the select list
-- advance in lockstep), so callers must pass them with equal lengths.
-- name: UpsertParticipantNames :exec
INSERT INTO participant_name (course_phase_id, course_participation_id, first_name, last_name)
SELECT sqlc.arg(course_phase_id)::uuid,
       unnest(sqlc.arg(course_participation_ids)::uuid[]),
       unnest(sqlc.arg(first_names)::text[]),
       unnest(sqlc.arg(last_names)::text[])
ON CONFLICT (course_phase_id, course_participation_id) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name  = EXCLUDED.last_name;
