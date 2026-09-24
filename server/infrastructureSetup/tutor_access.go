package infrastructureSetup

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func ensureTutorGroupAccess(git *gitlab.Client, introCourseGroupID, tutorsGroupID int64) error {
	group, _, err := git.Groups.GetGroup(introCourseGroupID, nil)
	if err != nil {
		return fmt.Errorf("inspect Introcourse group: %w", err)
	}
	for _, shared := range group.SharedWithGroups {
		if shared.GroupID == tutorsGroupID && shared.GroupAccessLevel >= int64(gitlab.DeveloperPermissions) {
			return nil
		}
	}
	_, _, err = git.Groups.ShareGroupWithGroup(introCourseGroupID, &gitlab.ShareGroupWithGroupOptions{
		GroupID:     gitlab.Ptr(tutorsGroupID),
		GroupAccess: gitlab.Ptr(gitlab.DeveloperPermissions),
	})
	if err != nil {
		return fmt.Errorf("grant tutor group Developer access: %w", err)
	}
	return nil
}
