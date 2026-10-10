package teamDTO

import (
	"encoding/json"
	"strings"

	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-sdk/promptTypes"
)

// GetTeamDTOFromDBModel turns a tutor and the students seated in their seats into a team.
// The team carries the tutor's ID and is named after them.
func GetTeamDTOFromDBModel(dbTeam db.GetTutorTeamsRow) (promptTypes.Team, error) {
	members := make([]promptTypes.Person, 0)
	if err := json.Unmarshal(dbTeam.TeamMembers, &members); err != nil {
		return promptTypes.Team{}, err
	}

	return promptTypes.Team{
		ID:      dbTeam.ID,
		Name:    strings.TrimSpace(dbTeam.FirstName + " " + dbTeam.LastName),
		Members: members,
		Tutors: []promptTypes.Person{
			{
				ID:        dbTeam.ID,
				FirstName: dbTeam.FirstName,
				LastName:  dbTeam.LastName,
			},
		},
	}, nil
}

func GetTeamDTOsFromDBModels(dbTeams []db.GetTutorTeamsRow) ([]promptTypes.Team, error) {
	teams := make([]promptTypes.Team, 0, len(dbTeams))
	for _, dbTeam := range dbTeams {
		team, err := GetTeamDTOFromDBModel(dbTeam)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}
