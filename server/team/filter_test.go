package team

import (
	"testing"

	"github.com/google/uuid"
	"github.com/prompt-edu/prompt-intro-course/server/team/teamDTO"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
	"github.com/stretchr/testify/assert"
)

func TestFilterTeamsByIDKeepsEveryTeamOfTheTutor(t *testing.T) {
	teamA, teamB, teamC := uuid.New(), uuid.New(), uuid.New()
	teams := []promptTypes.Team{{ID: teamA, Name: "A"}, {ID: teamB, Name: "B"}, {ID: teamC, Name: "C"}}

	assert.Equal(t, []promptTypes.Team{{ID: teamA, Name: "A"}, {ID: teamC, Name: "C"}}, filterTeamsByID(teams, []uuid.UUID{teamA, teamC}))
	assert.Empty(t, filterTeamsByID(teams, []uuid.UUID{uuid.New()}))
}

func TestFilterAllocationsByTeamKeepsEveryTeamOfTheTutor(t *testing.T) {
	teamA, teamB := uuid.New(), uuid.New()
	inA := teamDTO.AllocationWithParticipation{CourseParticipationID: uuid.New(), TeamAllocation: teamA}
	inB := teamDTO.AllocationWithParticipation{CourseParticipationID: uuid.New(), TeamAllocation: teamB}
	other := teamDTO.AllocationWithParticipation{CourseParticipationID: uuid.New(), TeamAllocation: uuid.New()}
	allocations := []teamDTO.AllocationWithParticipation{inA, other, inB}

	assert.Equal(t, []teamDTO.AllocationWithParticipation{inA, inB}, filterAllocationsByTeam(allocations, []uuid.UUID{teamA, teamB}))
	assert.Empty(t, filterAllocationsByTeam(allocations, []uuid.UUID{uuid.New()}))
}
