package seatPlanDTO

// ImportSeatAssignment represents a single row in the CSV import.
// All matching is done by name — IDs are resolved server-side.
type ImportSeatAssignment struct {
	SeatName        string `json:"seatName"`
	SeatMac         bool   `json:"seatMac"`
	AssignedStudent string `json:"assignedStudent"` // full name or "Unassigned"
	AssignedTutor   string `json:"assignedTutor"`   // full name or ""
	IsTutorSeat     bool   `json:"isTutorSeat"`
	PeerGroup       string `json:"peerGroup"` // e.g. "P1", "P2"
}

// ImportRequest is the payload for the CSV import endpoint.
type ImportRequest struct {
	Assignments []ImportSeatAssignment `json:"assignments"`
}

// ImportResultEntry reports the outcome for a single seat.
type ImportResultEntry struct {
	SeatName string `json:"seatName"`
	Success  bool   `json:"success"`
	Warning  string `json:"warning,omitempty"`
}

// ImportResult is the response from the CSV import endpoint.
type ImportResult struct {
	SeatsUpdated     int                 `json:"seatsUpdated"`
	PeerGroupsImported int               `json:"peerGroupsImported"`
	Warnings         []string            `json:"warnings,omitempty"`
	Details          []ImportResultEntry `json:"details,omitempty"`
}
