# iOS 2026–27 intro-course setup

## Release order

1. Merge and review the student template, daily issues, and `ci_cd/` files in [teaching-ios-material](https://gitlab.lrz.de/ase/ipraktikum/teaching-ios-material) before creating student repositories. The intro-course service reads that repository's `main` branch and caches the files in memory; restart it after a teaching-material change.
2. Merge this PROMPT change, then publish a new GitHub release only when production deployment is intended. Publishing a release triggers `.github/workflows/prod.yml`; changing `client/package.json` or merging a PR alone does not deploy production.
3. In PROMPT, assign each student a tutor. Run **Create infrastructure** (or **Check and repair infrastructure**), then create one ready student's repository and inspect its GitLab project name, files, executable exercise script, shared CI pipeline, and daily issues. Run a sample pipeline using ordinary student permissions and confirm that its lint policy can be fetched. Only then run the remaining batch. Students without a developer profile or GitLab username remain pending; rerun the batch after they complete it.
4. A repository request is marked successful only when project configuration and daily-issue creation finish. A failed request may have created part of a project; correct the error and rerun it. The stable project URL path is the TUM login; its visible name is `Student Name (TUM login)`.

## Provisional room plan

The current estimate is **47 students, six tutors, and eight students without Macs**. That is five tutor groups of eight students and one of seven. A workable starting split is Aquarium 16, iTüpferl 16, and Aurarium 15 students, with two tutors per room. This is a student-count target, not a claim about room capacity or the final physical seating chart.

Keep all eight students without Macs in Aquarium: four with each tutor. For each eight-student Aquarium group, place four usable Mac-equipped seats alternating with four seats for students bringing Macs, if the physical layout permits. Number actual usable seats in physical order (`Aquarium-01`, `Aquarium-02`, …) so consecutive numbers are adjacent. Reserve tutor seats separately if tutors need seats; confirm total capacity before uploading. The four Mac-equipped seats in a tutor group must be marked as Mac seats in PROMPT. For example, if its eight student seats are physically in one row, positions 01, 03, 05, and 07 could be Mac-equipped seats, with students bringing Macs at 02, 04, 06, and 08. Do not use this example as a seat map until the room is checked.

Upload a CSV with one unique seat name per line, using the exact room prefix and physical seat order. Mark which seats have Macs, assign the two tutors to their own contiguous seat blocks in each room, and mark any actual tutor seats. Review the room grid and download the assignment CSV before using the plan. For custom rooms, Smart Assign requires tutor ownership for every seat and a known Mac status for every student; it refuses to save a plan that leaves students unassigned or places a student without a Mac at a non-Mac seat. The legacy Rechnerhalle preset is only for a course actually held there.

Before finalizing assignments, confirm the official room spellings, usable seat counts, which seats contain Macs, the six tutor names, and the outstanding student's developer profile. The student count can then be adjusted between iTüpferl and Aurarium to match actual capacity.
