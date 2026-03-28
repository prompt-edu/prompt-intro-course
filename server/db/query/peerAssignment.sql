-- name: GetPeerAssignments :many
SELECT * FROM peer_assignment
WHERE course_phase_id = $1;

-- name: GetPeersForStudent :many
SELECT pa.peer_id, dp.gitlab_username, s.seat_name,
       t.gitlab_username AS tutor_gitlab_username
FROM peer_assignment pa
LEFT JOIN developer_profile dp
  ON pa.course_phase_id = dp.course_phase_id
  AND pa.peer_id = dp.course_participation_id
LEFT JOIN seat s
  ON pa.course_phase_id = s.course_phase_id
  AND pa.peer_id = s.assigned_student
LEFT JOIN tutor t
  ON s.course_phase_id = t.course_phase_id
  AND s.assigned_tutor = t.id
WHERE pa.course_phase_id = $1
  AND pa.student_id = $2;

-- name: GetReviewersForStudent :many
SELECT pa.student_id, dp.gitlab_username, s.seat_name,
       t.gitlab_username AS tutor_gitlab_username
FROM peer_assignment pa
LEFT JOIN developer_profile dp
  ON pa.course_phase_id = dp.course_phase_id
  AND pa.student_id = dp.course_participation_id
LEFT JOIN seat s
  ON pa.course_phase_id = s.course_phase_id
  AND pa.student_id = s.assigned_student
LEFT JOIN tutor t
  ON s.course_phase_id = t.course_phase_id
  AND s.assigned_tutor = t.id
WHERE pa.course_phase_id = $1
  AND pa.peer_id = $2;

-- name: CreatePeerAssignment :exec
INSERT INTO peer_assignment (course_phase_id, student_id, peer_id)
VALUES ($1, $2, $3)
ON CONFLICT ON CONSTRAINT peer_assignment_pkey DO NOTHING;

-- name: DeletePeerAssignments :exec
DELETE FROM peer_assignment
WHERE course_phase_id = $1;

