-- E2E Seed Data for Intro Course
-- course_phase_id: 4179d58a-d00d-4fa7-94a5-397bc69fab02
--
-- 6 Tutors, 56 Students, 89 Seats (Rechnerhalle layout), Peer Assignments
--
-- Tutor→Row mapping:
--   Alice (tutor-0) → R1        cap=10
--   Bob   (tutor-1) → R2        cap=8
--   Clara (tutor-2) → R3        cap=9
--   David (tutor-3) → R4,R5     cap=10  (Mac seats in R5)
--   Eva   (tutor-4) → R6,R7    cap=10  (Mac seats in R6)
--   Felix (tutor-5) → R8,R9    cap=9
--
-- Mac-needy students (has_macbook=false, 0-indexed): 5,8,13,18,23,29,33,37,41,45,50
--   → David gets indices 5,8,13,18,23,29
--   → Eva   gets indices 33,37,41,45,50
--
-- Peer groups use 3a+4b=n partitioning with bidirectional pairs.

BEGIN;

-- ============================================================
-- CLEANUP: remove existing data for this course phase
-- ============================================================
DELETE FROM peer_assignment   WHERE course_phase_id = '4179d58a-d00d-4fa7-94a5-397bc69fab02';
DELETE FROM seat              WHERE course_phase_id = '4179d58a-d00d-4fa7-94a5-397bc69fab02';
DELETE FROM developer_profile WHERE course_phase_id = '4179d58a-d00d-4fa7-94a5-397bc69fab02';
DELETE FROM tutor             WHERE course_phase_id = '4179d58a-d00d-4fa7-94a5-397bc69fab02';

-- ============================================================
-- TUTORS (6)
-- ============================================================
INSERT INTO tutor (course_phase_id, id, first_name, last_name, email, matriculation_number, university_login, gitlab_username) VALUES
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000001', 'Alice',  'Mueller',  'alice.mueller@example.com',  '030001', 'ga01abc', 'amueller'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000002', 'Bob',    'Schmidt',  'bob.schmidt@example.com',    '030002', 'ga02bcd', 'bschmidt'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000003', 'Clara',  'Weber',    'clara.weber@example.com',    '030003', 'ga03cde', 'cweber'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000004', 'David',  'Fischer',  'david.fischer@example.com',  '030004', 'ga04def', 'dfischer'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000005', 'Eva',    'Braun',    'eva.braun@example.com',      '030005', 'ga05efg', 'ebraun'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'a0000000-0000-0000-0000-000000000006', 'Felix',  'Wagner',   'felix.wagner@example.com',   '030006', 'ga06fgh', 'fwagner');

-- ============================================================
-- DEVELOPER PROFILES (56 students)
-- ============================================================
-- Student UUIDs: b0000000-0000-0000-0000-0000000000XX (XX = 01..56)
-- Mac-needy (has_macbook=false) at 0-indexed positions: 5,8,13,18,23,29,33,37,41,45,50
--   → student IDs: 06,09,14,19,24,30,34,38,42,46,51

INSERT INTO developer_profile (course_participation_id, course_phase_id, gitlab_username, apple_id, has_macbook, iphone_udid, ipad_udid, apple_watch_udid) VALUES
-- Student  0: Max Mueller        (has_macbook=true)
('b0000000-0000-0000-0000-000000000001', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'mmueller',      'max.mueller@icloud.com',      true,  NULL, NULL, NULL),
-- Student  1: Anna Schneider     (has_macbook=true)
('b0000000-0000-0000-0000-000000000002', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'aschneider',    'anna.schneider@icloud.com',   true,  NULL, NULL, NULL),
-- Student  2: Lukas Wagner       (has_macbook=true)
('b0000000-0000-0000-0000-000000000003', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lwagner',       'lukas.wagner@icloud.com',     true,  NULL, NULL, NULL),
-- Student  3: Sophie Fischer     (has_macbook=true)
('b0000000-0000-0000-0000-000000000004', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'sfischer',      'sophie.fischer@icloud.com',   true,  NULL, NULL, NULL),
-- Student  4: Leon Weber         (has_macbook=true)
('b0000000-0000-0000-0000-000000000005', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lweber',        'leon.weber@icloud.com',       true,  NULL, NULL, NULL),
-- Student  5: Emma Braun         (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000006', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'ebrauns',       'emma.braun@icloud.com',       false, NULL, NULL, NULL),
-- Student  6: Paul Hoffmann      (has_macbook=true)
('b0000000-0000-0000-0000-000000000007', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'phoffmann',     'paul.hoffmann@icloud.com',    true,  NULL, NULL, NULL),
-- Student  7: Marie Schulz       (has_macbook=true)
('b0000000-0000-0000-0000-000000000008', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'mschulz',       'marie.schulz@icloud.com',     true,  NULL, NULL, NULL),
-- Student  8: Jonas Koch         (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000009', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'jkoch',         'jonas.koch@icloud.com',       false, NULL, NULL, NULL),
-- Student  9: Tim Klein          (has_macbook=true)
('b0000000-0000-0000-0000-000000000010', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'tklein',        'tim.klein@icloud.com',        true,  NULL, NULL, NULL),
-- Student 10: Felix Groß         (has_macbook=true)
('b0000000-0000-0000-0000-000000000011', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'fgross',        'felix.gross@icloud.com',      true,  NULL, NULL, NULL),
-- Student 11: Hannah Bauer       (has_macbook=true)
('b0000000-0000-0000-0000-000000000012', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'hbauer',        'hannah.bauer@icloud.com',     true,  NULL, NULL, NULL),
-- Student 12: Lena Berger        (has_macbook=true)
('b0000000-0000-0000-0000-000000000013', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lberger',       'lena.berger@icloud.com',      true,  NULL, NULL, NULL),
-- Student 13: Tom Richter        (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000014', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'trichter',      'tom.richter@icloud.com',      false, NULL, NULL, NULL),
-- Student 14: Laura Krause       (has_macbook=true)
('b0000000-0000-0000-0000-000000000015', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lkrause',       'laura.krause@icloud.com',     true,  NULL, NULL, NULL),
-- Student 15: Nico Wolf          (has_macbook=true)
('b0000000-0000-0000-0000-000000000016', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'nwolf',         'nico.wolf@icloud.com',        true,  NULL, NULL, NULL),
-- Student 16: Mia Schmitt        (has_macbook=true)
('b0000000-0000-0000-0000-000000000017', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'mschmitt',      'mia.schmitt@icloud.com',      true,  NULL, NULL, NULL),
-- Student 17: Finn Neumann       (has_macbook=true)
('b0000000-0000-0000-0000-000000000018', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'fneumann',      'finn.neumann@icloud.com',     true,  NULL, NULL, NULL),
-- Student 18: Sara Schwarz       (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000019', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'sschwarz',      'sara.schwarz@icloud.com',     false, NULL, NULL, NULL),
-- Student 19: Eric Zimmermann    (has_macbook=true)
('b0000000-0000-0000-0000-000000000020', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'ezimmermann',   'eric.zimmermann@icloud.com',  true,  NULL, NULL, NULL),
-- Student 20: Robin Braun        (has_macbook=true)
('b0000000-0000-0000-0000-000000000021', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'rbraun',        'robin.braun@icloud.com',      true,  NULL, NULL, NULL),
-- Student 21: Jan Beck           (has_macbook=true)
('b0000000-0000-0000-0000-000000000022', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'jbeck',         'jan.beck@icloud.com',         true,  NULL, NULL, NULL),
-- Student 22: Fiona Keller       (has_macbook=true)
('b0000000-0000-0000-0000-000000000023', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'fkeller',       'fiona.keller@icloud.com',     true,  NULL, NULL, NULL),
-- Student 23: Henry Hartmann     (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000024', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'hhartmann',     'henry.hartmann@icloud.com',   false, NULL, NULL, NULL),
-- Student 24: Ben Lang           (has_macbook=true)
('b0000000-0000-0000-0000-000000000025', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'blang',         'ben.lang@icloud.com',         true,  NULL, NULL, NULL),
-- Student 25: Lisa Schäfer       (has_macbook=true)
('b0000000-0000-0000-0000-000000000026', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lschaefer',     'lisa.schaefer@icloud.com',    true,  NULL, NULL, NULL),
-- Student 26: Lea Werner         (has_macbook=true)
('b0000000-0000-0000-0000-000000000027', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lwerner',       'lea.werner@icloud.com',       true,  NULL, NULL, NULL),
-- Student 27: Lars Seidel        (has_macbook=true)
('b0000000-0000-0000-0000-000000000028', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lseidel',       'lars.seidel@icloud.com',      true,  NULL, NULL, NULL),
-- Student 28: Timo Meyer         (has_macbook=true)
('b0000000-0000-0000-0000-000000000029', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'tmeyer',        'timo.meyer@icloud.com',       true,  NULL, NULL, NULL),
-- Student 29: Julia Lange        (has_macbook=false) → David
('b0000000-0000-0000-0000-000000000030', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'jlange',        'julia.lange@icloud.com',      false, NULL, NULL, NULL),
-- Student 30: Nina Schmid        (has_macbook=true)
('b0000000-0000-0000-0000-000000000031', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'nschmid',       'nina.schmid@icloud.com',      true,  NULL, NULL, NULL),
-- Student 31: Alex Meier         (has_macbook=true)
('b0000000-0000-0000-0000-000000000032', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'ameier',        'alex.meier@icloud.com',       true,  NULL, NULL, NULL),
-- Student 32: Diana Krug         (has_macbook=true)
('b0000000-0000-0000-0000-000000000033', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'dkrug',         'diana.krug@icloud.com',       true,  NULL, NULL, NULL),
-- Student 33: Nora Hahn          (has_macbook=false) → Eva
('b0000000-0000-0000-0000-000000000034', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'nhahn',         'nora.hahn@icloud.com',        false, NULL, NULL, NULL),
-- Student 34: Jakob Kaiser        (has_macbook=true)
('b0000000-0000-0000-0000-000000000035', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'jkaiser',       'jakob.kaiser@icloud.com',     true,  NULL, NULL, NULL),
-- Student 35: Clara Weiß         (has_macbook=true)
('b0000000-0000-0000-0000-000000000036', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'cweiss',        'clara.weiss@icloud.com',      true,  NULL, NULL, NULL),
-- Student 36: Max König          (has_macbook=true)
('b0000000-0000-0000-0000-000000000037', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'mkoenig',       'max.koenig@icloud.com',       true,  NULL, NULL, NULL),
-- Student 37: Anne Frank         (has_macbook=false) → Eva
('b0000000-0000-0000-0000-000000000038', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'afrank',        'anne.frank@icloud.com',       false, NULL, NULL, NULL),
-- Student 38: Hugo Peters        (has_macbook=true)
('b0000000-0000-0000-0000-000000000039', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'hpeters',       'hugo.peters@icloud.com',      true,  NULL, NULL, NULL),
-- Student 39: Pia Brandt         (has_macbook=true)
('b0000000-0000-0000-0000-000000000040', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'pbrandt',       'pia.brandt@icloud.com',       true,  NULL, NULL, NULL),
-- Student 40: Cleo Ludwig        (has_macbook=true)
('b0000000-0000-0000-0000-000000000041', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'cludwig',       'cleo.ludwig@icloud.com',      true,  NULL, NULL, NULL),
-- Student 41: Oscar Sommer       (has_macbook=false) → Eva
('b0000000-0000-0000-0000-000000000042', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'osommer',       'oscar.sommer@icloud.com',     false, NULL, NULL, NULL),
-- Student 42: Ella Maier         (has_macbook=true)
('b0000000-0000-0000-0000-000000000043', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'emaier',        'ella.maier@icloud.com',       true,  NULL, NULL, NULL),
-- Student 43: Karl Wirth         (has_macbook=true)
('b0000000-0000-0000-0000-000000000044', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'kwirth',        'karl.wirth@icloud.com',       true,  NULL, NULL, NULL),
-- Student 44: Kurt Jung          (has_macbook=true)
('b0000000-0000-0000-0000-000000000045', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'kjung',         'kurt.jung@icloud.com',        true,  NULL, NULL, NULL),
-- Student 45: Eva Horn           (has_macbook=false) → Eva
('b0000000-0000-0000-0000-000000000046', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'ehorn',         'eva.horn@icloud.com',         false, NULL, NULL, NULL),
-- Student 46: Zoe Stein          (has_macbook=true)
('b0000000-0000-0000-0000-000000000047', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'zstein',        'zoe.stein@icloud.com',        true,  NULL, NULL, NULL),
-- Student 47: Sam Vogel          (has_macbook=true)
('b0000000-0000-0000-0000-000000000048', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'svogel',        'sam.vogel@icloud.com',        true,  NULL, NULL, NULL),
-- Student 48: Noah Fiedler       (has_macbook=true)
('b0000000-0000-0000-0000-000000000049', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'nfiedler',      'noah.fiedler@icloud.com',     true,  NULL, NULL, NULL),
-- Student 49: Ralf Krüger        (has_macbook=true)
('b0000000-0000-0000-0000-000000000050', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'rkrueger',      'ralf.krueger@icloud.com',     true,  NULL, NULL, NULL),
-- Student 50: Lara Koenig        (has_macbook=false) → Eva
('b0000000-0000-0000-0000-000000000051', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'lkoenig',       'lara.koenig@icloud.com',      false, NULL, NULL, NULL),
-- Student 51: Theo Günther       (has_macbook=true)
('b0000000-0000-0000-0000-000000000052', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'tguenther',     'theo.guenther@icloud.com',    true,  NULL, NULL, NULL),
-- Student 52: Peter Fuchs        (has_macbook=true)
('b0000000-0000-0000-0000-000000000053', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'pfuchs',        'peter.fuchs@icloud.com',      true,  NULL, NULL, NULL),
-- Student 53: Ida Becker         (has_macbook=true)
('b0000000-0000-0000-0000-000000000054', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'ibecker',       'ida.becker@icloud.com',       true,  NULL, NULL, NULL),
-- Student 54: Tina Wendt         (has_macbook=true)
('b0000000-0000-0000-0000-000000000055', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'twendt',        'tina.wendt@icloud.com',       true,  NULL, NULL, NULL),
-- Student 55: Vera Roth          (has_macbook=true)
('b0000000-0000-0000-0000-000000000056', '4179d58a-d00d-4fa7-94a5-397bc69fab02', 'vroth',         'vera.roth@icloud.com',        true,  NULL, NULL, NULL);

-- ============================================================
-- SEATS (89 total across 9 rows in Rechnerhalle layout)
-- ============================================================
-- Row→Tutor mapping:
--   R1 → Alice (tutor-0)   11 student seats + 1 tutor seat = 12
--   R2 → Bob   (tutor-1)    9 student seats + 1 tutor seat = 10
--   R3 → Clara (tutor-2)    9 student seats + 1 tutor seat = 10
--   R4 → David (tutor-3)   10 student seats (no tutor seat here, tutor seat in R4-10)
--                           Actually: 9 student seats + 1 tutor seat = 10
--   R5 → David (tutor-3)   10 student seats (Mac seats local 1-6)
--   R6 → Eva   (tutor-4)    8 total: 8 student seats (Mac seats local 1-6)
--   R7 → Eva   (tutor-4)    9 student seats + 1 tutor seat = 10
--   R8 → Felix (tutor-5)    9 student seats + 1 tutor seat = 10
--   R9 → Felix (tutor-5)    9 student seats
--
-- Student seat counts per tutor:
--   Alice: R1 has 11 student seats → need 10 students, 1 seat empty
--   Bob:   R2 has 9 student seats  → need 8 students, 1 seat empty
--   Clara: R3 has 9 student seats  → need 9 students, 0 empty
--   David: R4 has 9 student seats + R5 has 10 student seats = 19 → need 10, 9 empty
--   Eva:   R6 has 8 student seats + R7 has 9 student seats = 17 → need 10, 7 empty
--   Felix: R8 has 9 student seats + R9 has 9 student seats = 18 → need 9, 9 empty
--
-- Mac-needy students assigned to Mac seats first in their tutor group.
-- David's Mac-needy (6): placed on R5 local 1-6 (Mac seats)
-- David's non-Mac (4): placed on R4 local 1-4
-- Eva's Mac-needy (5): placed on R6 local 1-5 (Mac seats)
-- Eva's non-Mac (5): placed on R7 local 1-5

INSERT INTO seat (course_phase_id, seat_name, has_mac, device_id, assigned_student, assigned_tutor, is_tutor_seat) VALUES

-- ===================== ROW 1 (Alice, tutor-0) =====================
-- 12 seats: local 1-12, physical positions 1-12, names 1-1-1 through 1-1-12
-- Tutor seat: 1-1-12 (local 12)
-- Student seats: local 1-11 → assign Alice's 10 students to local 1-10, local 11 empty
-- Alice's students (non-Mac, indices 0-9 of non-Mac list):
--   idx 0 = student 01 (Max Mueller)
--   idx 1 = student 02 (Anna Schneider)
--   idx 2 = student 03 (Lukas Wagner)
--   idx 3 = student 04 (Sophie Fischer)
--   idx 4 = student 05 (Leon Weber)
--   idx 6 = student 07 (Paul Hoffmann)
--   idx 7 = student 08 (Marie Schulz)
--   idx 9 = student 10 (Tim Klein)
--   idx10 = student 11 (Felix Groß)
--   idx11 = student 12 (Hannah Bauer)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-1',  false, NULL, 'b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-2',  false, NULL, 'b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-3',  false, NULL, 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-4',  false, NULL, 'b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-5',  false, NULL, 'b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-6',  false, NULL, 'b0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-7',  false, NULL, 'b0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-8',  false, NULL, 'b0000000-0000-0000-0000-000000000010', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-9',  false, NULL, 'b0000000-0000-0000-0000-000000000011', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-10', false, NULL, 'b0000000-0000-0000-0000-000000000012', 'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-11', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000001', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-1-12', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000001', true),

-- ===================== ROW 2 (Bob, tutor-1) =====================
-- 10 seats: local 1-10, physical positions 1-5,7-11, names 1-2-1 through 1-2-10
-- Tutor seat: 1-2-10 (local 10, physical position 11)
-- Student seats: local 1-9 → assign Bob's 8 students to local 1-8, local 9 empty
-- Bob's students (non-Mac, from non-Mac list after Alice's 10):
--   idx12 = student 13 (Lena Berger)
--   idx14 = student 15 (Laura Krause)
--   idx15 = student 16 (Nico Wolf)
--   idx16 = student 17 (Mia Schmitt)
--   idx17 = student 18 (Finn Neumann)
--   idx19 = student 20 (Eric Zimmermann)
--   idx20 = student 21 (Robin Braun)
--   idx21 = student 22 (Jan Beck)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-1',  false, NULL, 'b0000000-0000-0000-0000-000000000013', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-2',  false, NULL, 'b0000000-0000-0000-0000-000000000015', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-3',  false, NULL, 'b0000000-0000-0000-0000-000000000016', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-4',  false, NULL, 'b0000000-0000-0000-0000-000000000017', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-5',  false, NULL, 'b0000000-0000-0000-0000-000000000018', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-6',  false, NULL, 'b0000000-0000-0000-0000-000000000020', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-7',  false, NULL, 'b0000000-0000-0000-0000-000000000021', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-8',  false, NULL, 'b0000000-0000-0000-0000-000000000022', 'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-9',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000002', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-2-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000002', true),

-- ===================== ROW 3 (Clara, tutor-2) =====================
-- 10 seats: local 1-10, physical positions 3-12, names 1-3-1 through 1-3-10
-- Tutor seat: 1-3-10 (local 10, physical position 12)
-- Student seats: local 1-9 → assign Clara's 9 students to local 1-9
-- Clara's students (non-Mac, from non-Mac list after Bob's 8):
--   idx22 = student 23 (Fiona Keller)
--   idx24 = student 25 (Ben Lang)
--   idx25 = student 26 (Lisa Schäfer)
--   idx26 = student 27 (Lea Werner)
--   idx27 = student 28 (Lars Seidel)
--   idx28 = student 29 (Timo Meyer)
--   idx30 = student 31 (Nina Schmid)
--   idx31 = student 32 (Alex Meier)
--   idx32 = student 33 (Diana Krug)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-1',  false, NULL, 'b0000000-0000-0000-0000-000000000023', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-2',  false, NULL, 'b0000000-0000-0000-0000-000000000025', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-3',  false, NULL, 'b0000000-0000-0000-0000-000000000026', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-4',  false, NULL, 'b0000000-0000-0000-0000-000000000027', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-5',  false, NULL, 'b0000000-0000-0000-0000-000000000028', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-6',  false, NULL, 'b0000000-0000-0000-0000-000000000029', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-7',  false, NULL, 'b0000000-0000-0000-0000-000000000031', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-8',  false, NULL, 'b0000000-0000-0000-0000-000000000032', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-9',  false, NULL, 'b0000000-0000-0000-0000-000000000033', 'a0000000-0000-0000-0000-000000000003', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-3-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000003', true),

-- ===================== ROW 4 (David, tutor-3) =====================
-- 10 seats: local 1-10, physical positions 3-12, names 1-4-1 through 1-4-10
-- Tutor seat: 1-4-10 (local 10, physical position 12)
-- Student seats: local 1-9 → assign David's 4 non-Mac students to local 1-4, rest empty
-- David's non-Mac students:
--   idx34 = student 35 (Jakob Kaiser)
--   idx35 = student 36 (Clara Weiß)
--   idx36 = student 37 (Max König)
--   idx38 = student 39 (Hugo Peters)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-1',  false, NULL, 'b0000000-0000-0000-0000-000000000035', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-2',  false, NULL, 'b0000000-0000-0000-0000-000000000036', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-3',  false, NULL, 'b0000000-0000-0000-0000-000000000037', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-4',  false, NULL, 'b0000000-0000-0000-0000-000000000039', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-5',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-6',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-7',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-8',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-9',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-4-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', true),

-- ===================== ROW 5 (David, tutor-3) =====================
-- 10 seats: local 1-10, physical positions 3-12, names 1-5-1 through 1-5-10
-- Mac seats: local 1-6 (has_mac=true)
-- No tutor seat in R5
-- David's Mac-needy students on Mac seats local 1-6:
--   idx 5 = student 06 (Emma Braun)
--   idx 8 = student 09 (Jonas Koch)
--   idx13 = student 14 (Tom Richter)
--   idx18 = student 19 (Sara Schwarz)
--   idx23 = student 24 (Henry Hartmann)
--   idx29 = student 30 (Julia Lange)
-- Remaining seats (local 7-10) are empty, non-Mac
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-1',  true,  NULL, 'b0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-2',  true,  NULL, 'b0000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-3',  true,  NULL, 'b0000000-0000-0000-0000-000000000014', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-4',  true,  NULL, 'b0000000-0000-0000-0000-000000000019', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-5',  true,  NULL, 'b0000000-0000-0000-0000-000000000024', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-6',  true,  NULL, 'b0000000-0000-0000-0000-000000000030', 'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-7',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-8',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-9',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-5-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000004', false),

-- ===================== ROW 6 (Eva, tutor-4) =====================
-- 8 seats: local 1-8, physical positions 3-5,7-11, names 1-6-1 through 1-6-8
-- Mac seats: local 1-6 (has_mac=true)
-- No tutor seat in R6
-- Eva's Mac-needy students on Mac seats local 1-5:
--   idx33 = student 34 (Nora Hahn)
--   idx37 = student 38 (Anne Frank)
--   idx41 = student 42 (Oscar Sommer)
--   idx45 = student 46 (Eva Horn)
--   idx50 = student 51 (Lara Koenig)
-- Mac seat local 6 is empty (has_mac=true but no student)
-- Non-Mac seats local 7-8 are empty
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-1',  true,  NULL, 'b0000000-0000-0000-0000-000000000034', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-2',  true,  NULL, 'b0000000-0000-0000-0000-000000000038', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-3',  true,  NULL, 'b0000000-0000-0000-0000-000000000042', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-4',  true,  NULL, 'b0000000-0000-0000-0000-000000000046', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-5',  true,  NULL, 'b0000000-0000-0000-0000-000000000051', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-6',  true,  NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-7',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-6-8',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),

-- ===================== ROW 7 (Eva, tutor-4) =====================
-- 10 seats: local 1-10, physical positions 3-12, names 1-7-1 through 1-7-10
-- Tutor seat: 1-7-10 (local 10, physical position 12)
-- Student seats: local 1-9 → assign Eva's 5 non-Mac students to local 1-5, rest empty
-- Eva's non-Mac students:
--   idx39 = student 40 (Pia Brandt)
--   idx40 = student 41 (Cleo Ludwig)
--   idx42 = student 43 (Ella Maier)
--   idx43 = student 44 (Karl Wirth)
--   idx44 = student 45 (Kurt Jung)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-1',  false, NULL, 'b0000000-0000-0000-0000-000000000040', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-2',  false, NULL, 'b0000000-0000-0000-0000-000000000041', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-3',  false, NULL, 'b0000000-0000-0000-0000-000000000043', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-4',  false, NULL, 'b0000000-0000-0000-0000-000000000044', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-5',  false, NULL, 'b0000000-0000-0000-0000-000000000045', 'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-6',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-7',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-8',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-9',  false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-7-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000005', true),

-- ===================== ROW 8 (Felix, tutor-5) =====================
-- 10 seats: local 1-10, physical positions 3-12, names 1-8-1 through 1-8-10
-- Tutor seat: 1-8-10 (local 10, physical position 12)
-- Student seats: local 1-9 → assign Felix's 9 students to local 1-9
-- Felix's students (all non-Mac):
--   idx46 = student 47 (Zoe Stein)
--   idx47 = student 48 (Sam Vogel)
--   idx48 = student 49 (Noah Fiedler)
--   idx49 = student 50 (Ralf Krüger)
--   idx51 = student 52 (Theo Günther)
--   idx52 = student 53 (Peter Fuchs)
--   idx53 = student 54 (Ida Becker)
--   idx54 = student 55 (Tina Wendt)
--   idx55 = student 56 (Vera Roth)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-1',  false, NULL, 'b0000000-0000-0000-0000-000000000047', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-2',  false, NULL, 'b0000000-0000-0000-0000-000000000048', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-3',  false, NULL, 'b0000000-0000-0000-0000-000000000049', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-4',  false, NULL, 'b0000000-0000-0000-0000-000000000050', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-5',  false, NULL, 'b0000000-0000-0000-0000-000000000052', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-6',  false, NULL, 'b0000000-0000-0000-0000-000000000053', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-7',  false, NULL, 'b0000000-0000-0000-0000-000000000054', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-8',  false, NULL, 'b0000000-0000-0000-0000-000000000055', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-9',  false, NULL, 'b0000000-0000-0000-0000-000000000056', 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-8-10', false, NULL, NULL,                                   'a0000000-0000-0000-0000-000000000006', true),

-- ===================== ROW 9 (Felix, tutor-5) =====================
-- 9 seats: local 1-9, physical positions 3-11, names 1-9-1 through 1-9-9
-- No tutor seat in R9
-- All seats empty (Felix's 9 students already placed in R8)
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-1',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-2',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-3',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-4',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-5',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-6',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-7',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-8',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', '1-9-9',  false, NULL, NULL, 'a0000000-0000-0000-0000-000000000006', false);

-- ============================================================
-- PEER ASSIGNMENTS
-- ============================================================
-- Partitioning: 3a + 4b = n
--   Alice (10): 3+3+4       Bob (8): 4+4
--   Clara  (9): 3+3+3       David (10): 3+3+4
--   Eva   (10): 3+3+4       Felix (9): 3+3+3
--
-- Students are listed in seat assignment order per tutor group.
-- Bidirectional pairs: for each group, every pair (A,B) produces (A,B) and (B,A).

INSERT INTO peer_assignment (course_phase_id, student_id, peer_id) VALUES

-- ===================== Alice's group (10 students) =====================
-- Alice students in seat order:
--   s01 = b...01 (Max Mueller)
--   s02 = b...02 (Anna Schneider)
--   s03 = b...03 (Lukas Wagner)
--   s04 = b...04 (Sophie Fischer)
--   s05 = b...05 (Leon Weber)
--   s07 = b...07 (Paul Hoffmann)
--   s08 = b...08 (Marie Schulz)
--   s10 = b...10 (Tim Klein)
--   s11 = b...11 (Felix Groß)
--   s12 = b...12 (Hannah Bauer)
--
-- Peer group A1 (3): s01, s02, s03
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000003'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000001'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000003'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000001'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000002'),
-- Peer group A2 (3): s04, s05, s07
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000005'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000007'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000004'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000007'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000004'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000005'),
-- Peer group A3 (4): s08, s10, s11, s12
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000010'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000011'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000012'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000008'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000011'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000012'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000008'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000010'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000011', 'b0000000-0000-0000-0000-000000000012'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000008'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000010'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000012', 'b0000000-0000-0000-0000-000000000011'),

-- ===================== Bob's group (8 students) =====================
-- Bob students in seat order:
--   s13 = b...13 (Lena Berger)
--   s15 = b...15 (Laura Krause)
--   s16 = b...16 (Nico Wolf)
--   s17 = b...17 (Mia Schmitt)
--   s18 = b...18 (Finn Neumann)
--   s20 = b...20 (Eric Zimmermann)
--   s21 = b...21 (Robin Braun)
--   s22 = b...22 (Jan Beck)
--
-- Peer group B1 (4): s13, s15, s16, s17
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000015'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000016'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000013', 'b0000000-0000-0000-0000-000000000017'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000013'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000016'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000015', 'b0000000-0000-0000-0000-000000000017'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000013'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000015'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000016', 'b0000000-0000-0000-0000-000000000017'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000017', 'b0000000-0000-0000-0000-000000000013'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000017', 'b0000000-0000-0000-0000-000000000015'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000017', 'b0000000-0000-0000-0000-000000000016'),
-- Peer group B2 (4): s18, s20, s21, s22
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000018', 'b0000000-0000-0000-0000-000000000020'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000018', 'b0000000-0000-0000-0000-000000000021'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000018', 'b0000000-0000-0000-0000-000000000022'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000020', 'b0000000-0000-0000-0000-000000000018'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000020', 'b0000000-0000-0000-0000-000000000021'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000020', 'b0000000-0000-0000-0000-000000000022'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000021', 'b0000000-0000-0000-0000-000000000018'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000021', 'b0000000-0000-0000-0000-000000000020'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000021', 'b0000000-0000-0000-0000-000000000022'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000022', 'b0000000-0000-0000-0000-000000000018'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000022', 'b0000000-0000-0000-0000-000000000020'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000022', 'b0000000-0000-0000-0000-000000000021'),

-- ===================== Clara's group (9 students) =====================
-- Clara students in seat order:
--   s23 = b...23 (Fiona Keller)
--   s25 = b...25 (Ben Lang)
--   s26 = b...26 (Lisa Schäfer)
--   s27 = b...27 (Lea Werner)
--   s28 = b...28 (Lars Seidel)
--   s29 = b...29 (Timo Meyer)
--   s31 = b...31 (Nina Schmid)
--   s32 = b...32 (Alex Meier)
--   s33 = b...33 (Diana Krug)
--
-- Peer group C1 (3): s23, s25, s26
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000023', 'b0000000-0000-0000-0000-000000000025'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000023', 'b0000000-0000-0000-0000-000000000026'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000025', 'b0000000-0000-0000-0000-000000000023'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000025', 'b0000000-0000-0000-0000-000000000026'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000026', 'b0000000-0000-0000-0000-000000000023'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000026', 'b0000000-0000-0000-0000-000000000025'),
-- Peer group C2 (3): s27, s28, s29
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000027', 'b0000000-0000-0000-0000-000000000028'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000027', 'b0000000-0000-0000-0000-000000000029'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000028', 'b0000000-0000-0000-0000-000000000027'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000028', 'b0000000-0000-0000-0000-000000000029'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000029', 'b0000000-0000-0000-0000-000000000027'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000029', 'b0000000-0000-0000-0000-000000000028'),
-- Peer group C3 (3): s31, s32, s33
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000031', 'b0000000-0000-0000-0000-000000000032'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000031', 'b0000000-0000-0000-0000-000000000033'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000032', 'b0000000-0000-0000-0000-000000000031'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000032', 'b0000000-0000-0000-0000-000000000033'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000033', 'b0000000-0000-0000-0000-000000000031'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000033', 'b0000000-0000-0000-0000-000000000032'),

-- ===================== David's group (10 students) =====================
-- David students: Mac-needy first (R5 Mac seats), then non-Mac (R4):
--   s06 = b...06 (Emma Braun)         - Mac seat R5-1
--   s09 = b...09 (Jonas Koch)         - Mac seat R5-2
--   s14 = b...14 (Tom Richter)        - Mac seat R5-3
--   s19 = b...19 (Sara Schwarz)       - Mac seat R5-4
--   s24 = b...24 (Henry Hartmann)     - Mac seat R5-5
--   s30 = b...30 (Julia Lange)        - Mac seat R5-6
--   s35 = b...35 (Jakob Kaiser)       - R4-1
--   s36 = b...36 (Clara Weiß)        - R4-2
--   s37 = b...37 (Max König)         - R4-3
--   s39 = b...39 (Hugo Peters)        - R4-4
--
-- Peer group D1 (3): s06, s09, s14
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000009'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000014'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000009', 'b0000000-0000-0000-0000-000000000006'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000009', 'b0000000-0000-0000-0000-000000000014'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000006'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000014', 'b0000000-0000-0000-0000-000000000009'),
-- Peer group D2 (3): s19, s24, s30
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000019', 'b0000000-0000-0000-0000-000000000024'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000019', 'b0000000-0000-0000-0000-000000000030'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000024', 'b0000000-0000-0000-0000-000000000019'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000024', 'b0000000-0000-0000-0000-000000000030'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000030', 'b0000000-0000-0000-0000-000000000019'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000030', 'b0000000-0000-0000-0000-000000000024'),
-- Peer group D3 (4): s35, s36, s37, s39
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000035', 'b0000000-0000-0000-0000-000000000036'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000035', 'b0000000-0000-0000-0000-000000000037'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000035', 'b0000000-0000-0000-0000-000000000039'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000036', 'b0000000-0000-0000-0000-000000000035'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000036', 'b0000000-0000-0000-0000-000000000037'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000036', 'b0000000-0000-0000-0000-000000000039'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000037', 'b0000000-0000-0000-0000-000000000035'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000037', 'b0000000-0000-0000-0000-000000000036'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000037', 'b0000000-0000-0000-0000-000000000039'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000039', 'b0000000-0000-0000-0000-000000000035'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000039', 'b0000000-0000-0000-0000-000000000036'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000039', 'b0000000-0000-0000-0000-000000000037'),

-- ===================== Eva's group (10 students) =====================
-- Eva students: Mac-needy first (R6 Mac seats), then non-Mac (R7):
--   s34 = b...34 (Nora Hahn)          - Mac seat R6-1
--   s38 = b...38 (Anne Frank)         - Mac seat R6-2
--   s42 = b...42 (Oscar Sommer)       - Mac seat R6-3
--   s46 = b...46 (Eva Horn)           - Mac seat R6-4
--   s51 = b...51 (Lara Koenig)        - Mac seat R6-5
--   s40 = b...40 (Pia Brandt)         - R7-1
--   s41 = b...41 (Cleo Ludwig)        - R7-2
--   s43 = b...43 (Ella Maier)         - R7-3
--   s44 = b...44 (Karl Wirth)         - R7-4
--   s45 = b...45 (Kurt Jung)          - R7-5
--
-- Peer group E1 (3): s34, s38, s42
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000034', 'b0000000-0000-0000-0000-000000000038'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000034', 'b0000000-0000-0000-0000-000000000042'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000038', 'b0000000-0000-0000-0000-000000000034'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000038', 'b0000000-0000-0000-0000-000000000042'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000042', 'b0000000-0000-0000-0000-000000000034'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000042', 'b0000000-0000-0000-0000-000000000038'),
-- Peer group E2 (3): s46, s51, s40
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000046', 'b0000000-0000-0000-0000-000000000051'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000046', 'b0000000-0000-0000-0000-000000000040'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000051', 'b0000000-0000-0000-0000-000000000046'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000051', 'b0000000-0000-0000-0000-000000000040'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000040', 'b0000000-0000-0000-0000-000000000046'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000040', 'b0000000-0000-0000-0000-000000000051'),
-- Peer group E3 (4): s41, s43, s44, s45
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000041', 'b0000000-0000-0000-0000-000000000043'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000041', 'b0000000-0000-0000-0000-000000000044'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000041', 'b0000000-0000-0000-0000-000000000045'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000043', 'b0000000-0000-0000-0000-000000000041'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000043', 'b0000000-0000-0000-0000-000000000044'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000043', 'b0000000-0000-0000-0000-000000000045'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000044', 'b0000000-0000-0000-0000-000000000041'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000044', 'b0000000-0000-0000-0000-000000000043'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000044', 'b0000000-0000-0000-0000-000000000045'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000045', 'b0000000-0000-0000-0000-000000000041'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000045', 'b0000000-0000-0000-0000-000000000043'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000045', 'b0000000-0000-0000-0000-000000000044'),

-- ===================== Felix's group (9 students) =====================
-- Felix students in seat order (all in R8):
--   s47 = b...47 (Zoe Stein)          - R8-1
--   s48 = b...48 (Sam Vogel)          - R8-2
--   s49 = b...49 (Noah Fiedler)       - R8-3
--   s50 = b...50 (Ralf Krüger)       - R8-4
--   s52 = b...52 (Theo Günther)      - R8-5
--   s53 = b...53 (Peter Fuchs)        - R8-6
--   s54 = b...54 (Ida Becker)         - R8-7
--   s55 = b...55 (Tina Wendt)         - R8-8
--   s56 = b...56 (Vera Roth)          - R8-9
--
-- Peer group F1 (3): s47, s48, s49
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000047', 'b0000000-0000-0000-0000-000000000048'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000047', 'b0000000-0000-0000-0000-000000000049'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000048', 'b0000000-0000-0000-0000-000000000047'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000048', 'b0000000-0000-0000-0000-000000000049'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000049', 'b0000000-0000-0000-0000-000000000047'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000049', 'b0000000-0000-0000-0000-000000000048'),
-- Peer group F2 (3): s50, s52, s53
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000050', 'b0000000-0000-0000-0000-000000000052'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000050', 'b0000000-0000-0000-0000-000000000053'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000052', 'b0000000-0000-0000-0000-000000000050'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000052', 'b0000000-0000-0000-0000-000000000053'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000053', 'b0000000-0000-0000-0000-000000000050'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000053', 'b0000000-0000-0000-0000-000000000052'),
-- Peer group F3 (3): s54, s55, s56
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000054', 'b0000000-0000-0000-0000-000000000055'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000054', 'b0000000-0000-0000-0000-000000000056'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000055', 'b0000000-0000-0000-0000-000000000054'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000055', 'b0000000-0000-0000-0000-000000000056'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000056', 'b0000000-0000-0000-0000-000000000054'),
('4179d58a-d00d-4fa7-94a5-397bc69fab02', 'b0000000-0000-0000-0000-000000000056', 'b0000000-0000-0000-0000-000000000055');

COMMIT;
