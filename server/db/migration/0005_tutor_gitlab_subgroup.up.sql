BEGIN;

CREATE TABLE tutor_gitlab_subgroup (
    course_phase_id   uuid NOT NULL,
    tutor_id          uuid NOT NULL,
    gitlab_group_id   bigint NOT NULL,
    gitlab_group_path text NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_phase_id, tutor_id),
    FOREIGN KEY (course_phase_id, tutor_id)
        REFERENCES tutor(course_phase_id, id) ON DELETE CASCADE
);

COMMIT;
