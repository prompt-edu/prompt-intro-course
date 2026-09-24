# Intro-course repository lifecycle

## Prepare a course

1. Import tutors in PROMPT, enter and check their LRZ GitLab usernames, and sync their course access. **Repository Setup** uses these records to populate the GitLab `tutors` group; it is safe to run again after a tutor correction.
2. Update and merge the student template, daily issues, and shared CI in [teaching-ios-material](https://gitlab.lrz.de/ase/ipraktikum/teaching-ios-material). **Repository Setup** reads one commit from its `main` branch for each setup or reset request. The page shows the commit it checked.
3. Run **Create or repair infrastructure**. It creates or verifies the semester groups, tutor access, shared CI project, and demo. It does **not** create student repositories. Review any reported issue before continuing.
4. After a teaching-material change, run **Create or repair infrastructure** to update shared CI, then **Reset demo** to obtain the new starter files and issues. Test `Introcourse/demo` as an instructor: generate and build the app, work through the issues, create a branch and merge request, request a review, and inspect the pipeline. GitLab's **Course work** board uses the inherited Open, In Progress, In Review, Blocked, and Done statuses. Exercise branches and merge requests make the demo intentionally unready for student initialization.
5. Use **Reset demo** again after each test. PROMPT pins the selected source commit, renames the old demo, creates a fresh one, and moves the old project into the private `demo-archives` subgroup. Its archive link remains available for investigation. Repeat this cycle until the clean demo, CI, access, and issues all pass the live readiness check.
6. Finalize the roster, excluding failed participants. Assign each student a seat and tutor; generate and review peer groups. Check each student's developer profile and GitLab username. The student-repository dialog reports missing seats, tutors, and peer groups before it allows a batch.
7. Only after the clean demo and a controlled student-style review test pass, initialize the student repositories. The batch uses the verified teaching-material commit and skips students who still lack required profile information. A pending student can be initialized later.

The stable student project URL path is the university login. The visible project name includes the student's name and login. The owner and tutor group can merge through reviewed merge requests; peers in the tutor group have Developer access for reviews and **Request changes**. One tutor approval is required, CI and discussions must pass, and direct pushes to `main` are blocked. GitLab applies the most permissive matching branch rule, so setup fails if a wildcard rule conflicts with `main` protection.

## Operations and recovery

- **Create or repair infrastructure** fills missing setup. It does not silently overwrite edited demo files or issue text. Reset the demo to apply new teaching material.
- A failed student setup may leave a partial repository. Investigate the error and retry the individual student after confirming its project path and tutor assignment. Do not reassign a student's tutor after repository creation without planning a GitLab project move; changing the tutor changes the destination subgroup.
- The demo proves material, CI, and tutor setup, but cannot prove student and peer permissions. Use a controlled student-style repository for that permission test; do not use real student repositories as smoke tests.
- Archive projects are retained for audit and repeated experiments. Review and remove old archives only after their results are no longer needed. Reset itself does not delete them.

## Physical seating

Keep the roster, room capacities, tutor coverage, and device needs in an access-controlled course plan. Upload actual physical seat names, mark Mac-equipped seats, and assign tutor ownership before running Smart Assign. Review and export the result. The Rechnerhalle preset is only for a course actually held there.
