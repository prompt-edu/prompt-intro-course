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

// Tutors retain Developer access but can merge reviewed changes into demo main.
// Sharing the project directly is required for a group-specific branch rule.
func ensureDemoTutorMergeAccess(git *gitlab.Client, projectID, tutorsGroupID int64) error {
	if err := ensureProjectSharedWithGroupAtLeast(git, projectID, tutorsGroupID, gitlab.DeveloperPermissions); err != nil {
		return fmt.Errorf("share demo with tutor group: %w", err)
	}
	branch, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, "main")
	if err != nil {
		return fmt.Errorf("inspect demo main: %w", err)
	}
	if demoMainBranchProtectionMatches(branch, tutorsGroupID) {
		return nil
	}
	if !mainBranchProtectionMatches(branch, 0) {
		return fmt.Errorf("secure demo main before granting tutor merge access")
	}
	branch, _, err = git.ProtectedBranches.UpdateProtectedBranch(projectID, "main", &gitlab.UpdateProtectedBranchOptions{
		AllowedToMerge: gitlab.Ptr([]*gitlab.BranchPermissionOptions{{GroupID: gitlab.Ptr(tutorsGroupID)}}),
	})
	if err != nil {
		return fmt.Errorf("allow tutors to merge demo changes: %w", err)
	}
	if !demoMainBranchProtectionMatches(branch, tutorsGroupID) {
		return fmt.Errorf("GitLab did not confirm tutor merge access and demo push restrictions")
	}
	return nil
}

func demoMainBranchProtectionMatches(branch *gitlab.ProtectedBranch, tutorsGroupID int64) bool {
	if branch == nil || tutorsGroupID == 0 {
		return false
	}
	base := *branch
	base.MergeAccessLevels = nil
	tutorGroupPresent := false
	for _, access := range branch.MergeAccessLevels {
		if access.GroupID == tutorsGroupID && access.UserID == 0 && !tutorGroupPresent {
			tutorGroupPresent = true
		} else {
			base.MergeAccessLevels = append(base.MergeAccessLevels, access)
		}
	}
	return tutorGroupPresent && mainBranchProtectionMatches(&base, 0)
}
