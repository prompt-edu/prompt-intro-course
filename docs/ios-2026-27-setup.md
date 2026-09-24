# iOS 2026–27 intro-course setup

## Release order

1. Merge and review the student template, daily issues, and `ci_cd/` files in [teaching-ios-material](https://gitlab.lrz.de/ase/ipraktikum/teaching-ios-material) before creating student repositories. The intro-course service reads that repository's `main` branch and caches the files in memory; restart it after a teaching-material change.
2. Merge this PROMPT change, then publish a new GitHub release only when production deployment is intended. Publishing a release triggers `.github/workflows/prod.yml`; changing `client/package.json` or merging a PR alone does not deploy production.
3. In PROMPT, run **Check GitLab usernames** on Developer Profiles and compare each GitLab account name with the student. Failed participants are excluded from this roster and from seat assignment. Run **Create infrastructure** (or **Check and repair infrastructure**) first and inspect its `Introcourse/demo` project: files, executable exercise script, shared CI pipeline, daily issues, inherited tutor access, and approval rule. Run a sample pipeline and draft MR. Import tutors and use **Sync to Keycloak**; confirm there are no warnings before relying on tutor access to Intro Assessment. Do not create student repositories until the course team is ready to initialize them. A student without a verified GitLab username remains pending and can be initialized later.
4. A repository request is marked successful only when project configuration and daily-issue creation finish. A failed request may have created part of a project; correct the error and rerun it. The stable project URL path is the TUM login; its visible name is `Student Name - TUM login` (with unsupported punctuation replaced by spaces).

## Room plan

Keep the current roster, room capacities, tutor coverage, and individual device needs in the course team's access-controlled planning document. Exclude failed participants from seating. Confirm each physical Mac-equipped station before marking seats or assigning students.

Upload a CSV with one unique seat name per line, using the actual room and physical seat order. Assign tutor ownership to every seat and mark tutor seats separately. Review the room grid and download the assignment CSV before using the plan. For custom rooms, Smart Assign needs tutor ownership for every seat and a known Mac status for every student; it refuses a plan that leaves students unassigned or places a student without a Mac at a non-Mac seat. The legacy Rechnerhalle preset is only for a course actually held there.

## Resetting the demo after a smoke test

The demo's `main` branch is the clean template; keep experimental commits on branches and use draft MRs. To start over with a new demo project, first preserve the old project by renaming its GitLab path to a unique archive name (for example, `demo-smoke-2026-09-24`). Then use PROMPT's **Check and repair infrastructure** action to recreate `Introcourse/demo` from the current teaching materials. Inspect the new project's 12 daily issues, shared CI, branch protection, tutor access, and approval rule before continuing. Do not delete the archived project until its test results are no longer needed; GitLab project deletion may be delayed and is not required to recreate `demo` after the path has been freed. This operation concerns only the demo and does not create student repositories.
