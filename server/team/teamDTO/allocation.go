package teamDTO

import (
	"github.com/google/uuid"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
)

type Allocation struct {
	TeamAllocation uuid.UUID `json:"teamAllocation"`
}

type AllocationWithParticipation struct {
	CourseParticipationID uuid.UUID `json:"courseParticipationID"`
	TeamAllocation        uuid.UUID `json:"teamAllocation"`
}

func GetAllocationDTOsFromDBModels(dbAllocations []db.GetTeamAllocationsRow) []AllocationWithParticipation {
	allocations := make([]AllocationWithParticipation, 0, len(dbAllocations))
	for _, allocation := range dbAllocations {
		allocations = append(allocations, AllocationWithParticipation{
			CourseParticipationID: allocation.CourseParticipationID,
			TeamAllocation:        allocation.TeamID,
		})
	}
	return allocations
}
