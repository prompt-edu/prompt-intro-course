--
-- Intro-course additions to the core seed.
--
-- 10_core_seed.sql is a VERBATIM copy of prompt2/e2e/seed/e2e_seed.sql so it can
-- be refreshed with a plain file copy; everything intro-course specific lives
-- here. Both files are mounted into the core db's /docker-entrypoint-initdb.d and
-- run in filename order, so the tables, enums, and FK constraints created by the
-- dump already exist when this file runs.
--
-- Rows are inserted in FK order: phase type -> DTO -> phase -> graph edge ->
-- students -> course participations -> course phase participations.
--

--
-- The phase type. The name must be EXACTLY 'Intro Course Developer': that string is
-- the key in the core client's PhaseRouterMapping / PhaseSidebarMapping (so the
-- Module Federation remote is selected) and the name core's
-- coursePhaseType.initIntroCourseDeveloper matches on at startup. Because the row
-- already exists, core skips creating it -- which is the point: left to core it
-- would mint a RANDOM uuid and the seeded course_phase below could not reference
-- it. {CORE_HOST} is substituted by core at read time and must match the e2e nginx
-- proxy (/intro-course/api -> server-intro-course).
--
INSERT INTO public.course_phase_type VALUES ('c6666666-6666-6666-6666-666666666666', 'Intro Course Developer', false, '{CORE_HOST}/intro-course/api', 'Intro course phase: developer profiles, seat plan, tutors, and peer review groups.');

--
-- Mirrors core's InsertProvidedOutputDevices (skipped at startup because the phase
-- type above already exists). Downstream phases (team allocation) consume this DTO.
--
INSERT INTO public.course_phase_type_participation_provided_output_dto VALUES ('d1000010-0000-4000-8000-000000000010', 'c6666666-6666-6666-6666-666666666666', 'devices', 1, '/devices', '{"type": "array", "items": {"type": "string", "enum": ["IPhone", "IPad", "MacBook", "AppleWatch"]}}');

--
-- The intro course phase on iPraktikumFull. The id is fixed to the one
-- server/database_dumps/e2e_seed.sql is built around, so the intro-course db seed
-- needs no rewriting.
--
INSERT INTO public.course_phase VALUES ('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'c0000001-0000-0000-0000-000000000001', 'Intro Course', '{}', false, 'c6666666-6666-6666-6666-666666666666', '{}');

--
-- Append the phase to the TAIL of the iPraktikumFull chain (currently Certificate).
-- course_phase_graph has UNIQUE constraints on both columns, so the graph is a
-- chain -- never branch off an earlier phase.
--
INSERT INTO public.course_phase_graph VALUES ('d000000d-0000-0000-0000-00000000000d', '4179d58a-d00d-4fa7-94a5-397bc69fab02');

--
-- 56 background students matching the developer profiles, seats, and peer groups in
-- server/database_dumps/e2e_seed.sql. Every lecturer page resolves names through
-- core's getCoursePhaseParticipations, so without these rows the seat grid, the
-- developer-profile table, and the peer-group list all render blank cells.
--
-- course_participation ids are b0000000-0000-0000-0000-0000000000NN (NN = 01..56),
-- the exact ids the intro-course seed references.
--
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000001', 'Max', 'Mueller', 'max.mueller@example.com', '020001', 'ic01aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000002', 'Anna', 'Schneider', 'anna.schneider@example.com', '020002', 'ic02aaa', true, 'female', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000003', 'Lukas', 'Wagner', 'lukas.wagner@example.com', '020003', 'ic03aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000004', 'Sophie', 'Fischer', 'sophie.fischer@example.com', '020004', 'ic04aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000005', 'Leon', 'Weber', 'leon.weber@example.com', '020005', 'ic05aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000006', 'Emma', 'Braun', 'emma.braun@example.com', '020006', 'ic06aaa', true, 'female', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000007', 'Paul', 'Hoffmann', 'paul.hoffmann@example.com', '020007', 'ic07aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000008', 'Marie', 'Schulz', 'marie.schulz@example.com', '020008', 'ic08aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000009', 'Jonas', 'Koch', 'jonas.koch@example.com', '020009', 'ic09aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000010', 'Tim', 'Klein', 'tim.klein@example.com', '020010', 'ic10aaa', true, 'female', 'DE', 'Computer Science', 'master', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000011', 'Felix', 'Groß', 'felix.gross@example.com', '020011', 'ic11aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000012', 'Hannah', 'Bauer', 'hannah.bauer@example.com', '020012', 'ic12aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000013', 'Lena', 'Berger', 'lena.berger@example.com', '020013', 'ic13aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000014', 'Tom', 'Richter', 'tom.richter@example.com', '020014', 'ic14aaa', true, 'female', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000015', 'Laura', 'Krause', 'laura.krause@example.com', '020015', 'ic15aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000016', 'Nico', 'Wolf', 'nico.wolf@example.com', '020016', 'ic16aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000017', 'Mia', 'Schmitt', 'mia.schmitt@example.com', '020017', 'ic17aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000018', 'Finn', 'Neumann', 'finn.neumann@example.com', '020018', 'ic18aaa', true, 'female', 'DE', 'Computer Science', 'master', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000019', 'Sara', 'Schwarz', 'sara.schwarz@example.com', '020019', 'ic19aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000020', 'Eric', 'Zimmermann', 'eric.zimmermann@example.com', '020020', 'ic20aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000021', 'Robin', 'Braun', 'robin.braun@example.com', '020021', 'ic21aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000022', 'Jan', 'Beck', 'jan.beck@example.com', '020022', 'ic22aaa', true, 'female', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000023', 'Fiona', 'Keller', 'fiona.keller@example.com', '020023', 'ic23aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000024', 'Henry', 'Hartmann', 'henry.hartmann@example.com', '020024', 'ic24aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000025', 'Ben', 'Lang', 'ben.lang@example.com', '020025', 'ic25aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000026', 'Lisa', 'Schäfer', 'lisa.schaefer@example.com', '020026', 'ic26aaa', true, 'female', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000027', 'Lea', 'Werner', 'lea.werner@example.com', '020027', 'ic27aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000028', 'Lars', 'Seidel', 'lars.seidel@example.com', '020028', 'ic28aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000029', 'Timo', 'Meyer', 'timo.meyer@example.com', '020029', 'ic29aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000030', 'Julia', 'Lange', 'julia.lange@example.com', '020030', 'ic30aaa', true, 'female', 'DE', 'Computer Science', 'master', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000031', 'Nina', 'Schmid', 'nina.schmid@example.com', '020031', 'ic31aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000032', 'Alex', 'Meier', 'alex.meier@example.com', '020032', 'ic32aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000033', 'Diana', 'Krug', 'diana.krug@example.com', '020033', 'ic33aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000034', 'Nora', 'Hahn', 'nora.hahn@example.com', '020034', 'ic34aaa', true, 'female', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000035', 'Jakob', 'Kaiser', 'jakob.kaiser@example.com', '020035', 'ic35aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000036', 'Clara', 'Weiß', 'clara.weiss@example.com', '020036', 'ic36aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000037', 'Max', 'König', 'max.koenig@example.com', '020037', 'ic37aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000038', 'Anne', 'Frank', 'anne.frank@example.com', '020038', 'ic38aaa', true, 'female', 'DE', 'Computer Science', 'master', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000039', 'Hugo', 'Peters', 'hugo.peters@example.com', '020039', 'ic39aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000040', 'Pia', 'Brandt', 'pia.brandt@example.com', '020040', 'ic40aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000041', 'Cleo', 'Ludwig', 'cleo.ludwig@example.com', '020041', 'ic41aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000042', 'Oscar', 'Sommer', 'oscar.sommer@example.com', '020042', 'ic42aaa', true, 'female', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000043', 'Ella', 'Maier', 'ella.maier@example.com', '020043', 'ic43aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000044', 'Karl', 'Wirth', 'karl.wirth@example.com', '020044', 'ic44aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000045', 'Kurt', 'Jung', 'kurt.jung@example.com', '020045', 'ic45aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000046', 'Eva', 'Horn', 'eva.horn@example.com', '020046', 'ic46aaa', true, 'female', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000047', 'Zoe', 'Stein', 'zoe.stein@example.com', '020047', 'ic47aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000048', 'Sam', 'Vogel', 'sam.vogel@example.com', '020048', 'ic48aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000049', 'Noah', 'Fiedler', 'noah.fiedler@example.com', '020049', 'ic49aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000050', 'Ralf', 'Krüger', 'ralf.krueger@example.com', '020050', 'ic50aaa', true, 'female', 'DE', 'Computer Science', 'master', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000051', 'Lara', 'Koenig', 'lara.koenig@example.com', '020051', 'ic51aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 4, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000052', 'Theo', 'Günther', 'theo.guenther@example.com', '020052', 'ic52aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 5, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000053', 'Peter', 'Fuchs', 'peter.fuchs@example.com', '020053', 'ic53aaa', true, 'male', 'DE', 'Computer Science', 'bachelor', 6, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000054', 'Ida', 'Becker', 'ida.becker@example.com', '020054', 'ic54aaa', true, 'female', 'DE', 'Computer Science', 'master', 7, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000055', 'Tina', 'Wendt', 'tina.wendt@example.com', '020055', 'ic55aaa', true, 'diverse', 'DE', 'Computer Science', 'bachelor', 3, '2025-04-01 09:00:00');
INSERT INTO public.student VALUES ('b1000000-0000-4000-8000-000000000056', 'Vera', 'Roth', 'vera.roth@example.com', '020056', 'ic56aaa', true, 'prefer_not_to_say', 'DE', 'Computer Science', 'master', 4, '2025-04-01 09:00:00');

INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000001', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000001');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000002', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000002');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000003', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000003');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000004', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000004');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000005', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000005');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000006', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000006');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000007', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000007');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000008', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000008');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000009', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000009');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000010', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000010');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000011', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000011');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000012', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000012');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000013', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000013');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000014', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000014');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000015', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000015');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000016', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000016');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000017', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000017');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000018', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000018');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000019', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000019');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000020', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000020');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000021', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000021');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000022', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000022');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000023', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000023');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000024', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000024');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000025', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000025');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000026', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000026');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000027', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000027');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000028', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000028');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000029', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000029');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000030', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000030');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000031', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000031');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000032', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000032');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000033', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000033');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000034', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000034');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000035', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000035');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000036', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000036');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000037', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000037');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000038', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000038');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000039', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000039');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000040', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000040');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000041', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000041');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000042', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000042');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000043', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000043');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000044', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000044');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000045', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000045');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000046', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000046');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000047', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000047');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000048', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000048');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000049', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000049');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000050', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000050');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000051', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000051');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000052', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000052');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000053', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000053');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000054', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000054');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000055', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000055');
INSERT INTO public.course_participation VALUES ('b0000000-0000-0000-0000-000000000056', 'c0000001-0000-0000-0000-000000000001', 'b1000000-0000-4000-8000-000000000056');

INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000001', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000002', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000003', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000004', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000005', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000006', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000007', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000008', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000009', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000010', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000011', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000012', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000013', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000014', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000015', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000016', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000017', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000018', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000019', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000020', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000021', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000022', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000023', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000024', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000025', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000026', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000027', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000028', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000029', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000030', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000031', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000032', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000033', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000034', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000035', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000036', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000037', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000038', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000039', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000040', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000041', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000042', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000043', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000044', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000045', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000046', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000047', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000048', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000049', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000050', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000051', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000052', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000053', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000054', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000055', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('b0000000-0000-0000-0000-000000000056', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');

--
-- The two Keycloak student users (see e2e/keycloak/realm.json): Stan (`student`) and
-- Selma (`student2`). They already have a course_participation on iPraktikumFull --
-- reuse it, do NOT add a second one, since course_participation is UNIQUE on
-- (course_id, student_id). Without an own participation on this phase the student
-- routes 404 into UnauthorizedPage.
--
-- Stan is deliberately left WITHOUT a developer_profile in the intro-course seed so
-- the student journey can submit the survey; Selma has a profile, a seat, and peers.
--
INSERT INTO public.course_phase_participation VALUES ('a0000001-0000-0000-0000-000000000001', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
INSERT INTO public.course_phase_participation VALUES ('ca000008-0000-4000-8000-000000000008', '4179d58a-d00d-4fa7-94a5-397bc69fab02', '{}', 'not_assessed', '2025-04-01 09:00:00', '{}');
