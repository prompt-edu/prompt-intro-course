package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// StudentRecord holds one row from the students CSV.
type StudentRecord struct {
	FirstName            string
	LastName             string
	Email                string
	MatriculationNumber  string
	UniversityLogin      string
	HasUniversityAccount bool
	CourseParticipationID string
	Gender               string // "MALE" or "FEMALE"
	PassStatus           string // "PASSED" or "FAILED"
}

// StudentLoader reads and caches student data from a semicolon-delimited CSV.
type StudentLoader struct {
	path     string
	Students []StudentRecord
}

// NewStudentLoader creates a loader for the given CSV path.
func NewStudentLoader(path string) *StudentLoader {
	return &StudentLoader{path: path}
}

// Load reads the CSV file and populates the Students slice.
func (sl *StudentLoader) Load() error {
	f, err := os.Open(sl.path)
	if err != nil {
		return fmt.Errorf("open %s: %w", sl.path, err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	reader.Comma = ';'
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) < 2 {
		log.Warn("Students CSV has no data rows")
		sl.Students = []StudentRecord{}
		return nil
	}

	// Skip header row.
	sl.Students = make([]StudentRecord, 0, len(records)-1)
	for i, row := range records[1:] {
		if len(row) < 9 {
			log.Warnf("Skipping CSV row %d: only %d columns", i+2, len(row))
			continue
		}
		sr := StudentRecord{
			FirstName:            clean(row[0]),
			LastName:             clean(row[1]),
			Email:                clean(row[2]),
			MatriculationNumber:  clean(row[3]),
			UniversityLogin:      clean(row[4]),
			HasUniversityAccount: strings.EqualFold(clean(row[5]), "true"),
			CourseParticipationID: clean(row[6]),
			Gender:               mapGender(clean(row[7])),
			PassStatus:           mapPassStatus(clean(row[8])),
		}
		sl.Students = append(sl.Students, sr)
	}
	return nil
}

// FindByParticipationID returns the student record with the given ID, or nil.
func (sl *StudentLoader) FindByParticipationID(id string) *StudentRecord {
	for i := range sl.Students {
		if sl.Students[i].CourseParticipationID == id {
			return &sl.Students[i]
		}
	}
	return nil
}

// clean strips surrounding quotes and whitespace.
func clean(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	return s
}

func mapGender(g string) string {
	switch strings.ToLower(g) {
	case "male":
		return "MALE"
	case "female":
		return "FEMALE"
	default:
		return strings.ToUpper(g)
	}
}

func mapPassStatus(s string) string {
	switch strings.ToLower(s) {
	case "passed":
		return "PASSED"
	case "failed":
		return "FAILED"
	default:
		return strings.ToUpper(s)
	}
}
