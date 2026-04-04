package peerAssignment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	"github.com/prompt-edu/prompt-intro-course/server/peerAssignment/peerAssignmentDTO"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const peerReviewRuleName = "Peer Review"

// SyncPeerAssignmentsToGitlab adds peers as Reporter members and creates
// "Peer Review" approval rules on each student's GitLab project.
func SyncPeerAssignmentsToGitlab(ctx context.Context, coursePhaseID uuid.UUID, semesterTag string) ([]peerAssignmentDTO.SyncResult, error) {
	svc := PeerAssignmentServiceSingleton
	if svc.gitlabClient == nil {
		return nil, gitlabutil.ErrClientNotInitialized
	}

	// Use a generous timeout for the full sync operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	dbCtx, dbCancel := db.GetTimeoutContext(ctx)
	defer dbCancel()
	assignments, err := svc.queries.GetPeerAssignments(dbCtx, coursePhaseID)
	if err != nil {
		return nil, fmt.Errorf("get peer assignments: %w", err)
	}

	// Cache user lookups to avoid redundant GitLab API calls
	userCache := make(map[string]*gitlab.User)

	var results []peerAssignmentDTO.SyncResult

	for _, assignment := range assignments {
		result := peerAssignmentDTO.SyncResult{
			StudentID: assignment.StudentID,
			PeerID:    assignment.PeerID,
		}

		err := syncSinglePeerAccess(ctx, svc, coursePhaseID, assignment.StudentID, assignment.PeerID, semesterTag, userCache)
		if err != nil {
			log.WithFields(log.Fields{
				"studentID": assignment.StudentID,
				"peerID":    assignment.PeerID,
				"error":     err,
			}).Warn("Failed to sync peer access to GitLab")
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	return results, nil
}

// syncSinglePeerAccess grants reviewer (studentID) Reporter access to the
// reviewee's (peerID) GitLab project and ensures a "Peer Review" approval rule.
func syncSinglePeerAccess(ctx context.Context, svc *PeerAssignmentService, coursePhaseID uuid.UUID, reviewerID, revieweeID uuid.UUID, semesterTag string, userCache map[string]*gitlab.User) error {
	git := svc.gitlabClient

	// 1. Look up reviewer's GitLab username
	reviewerProfile, err := svc.queries.GetDeveloperProfileByCourseParticipationID(ctx, db.GetDeveloperProfileByCourseParticipationIDParams{
		CourseParticipationID: reviewerID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get reviewer profile: %w", err)
	}
	if reviewerProfile.GitlabUsername == "" {
		return fmt.Errorf("reviewer %s has no GitLab username", reviewerID)
	}

	// 2. Look up reviewee's GitLab username and their tutor
	revieweeProfile, err := svc.queries.GetDeveloperProfileByCourseParticipationID(ctx, db.GetDeveloperProfileByCourseParticipationIDParams{
		CourseParticipationID: revieweeID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get reviewee profile: %w", err)
	}
	if revieweeProfile.GitlabUsername == "" {
		return fmt.Errorf("reviewee %s has no GitLab username", revieweeID)
	}

	// 3. Get reviewee's assigned tutor for path resolution
	tutor, err := svc.queries.GetAssignedTutor(ctx, db.GetAssignedTutorParams{
		AssignedStudent: pgtype.UUID{Bytes: revieweeID, Valid: true},
		CoursePhaseID:   coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get tutor for reviewee: %w", err)
	}
	if !tutor.GitlabUsername.Valid || tutor.GitlabUsername.String == "" {
		return fmt.Errorf("tutor for reviewee %s has no GitLab username", revieweeID)
	}

	// 4. Resolve reviewer GitLab user ID (cached for the sync run)
	reviewerGitlabUser, err := getCachedUser(git, reviewerProfile.GitlabUsername, userCache)
	if err != nil {
		return fmt.Errorf("resolve reviewer GitLab user: %w", err)
	}

	// Also resolve reviewee GitLab user for the fallback search below
	revieweeGitlabUser, err := getCachedUser(git, revieweeProfile.GitlabUsername, userCache)
	if err != nil {
		return fmt.Errorf("resolve reviewee GitLab user: %w", err)
	}

	// 5. Find reviewee's project by convention path
	// semesterTag is passed as-is (case must match what infrastructure setup used)
	projectPath := fmt.Sprintf("ase/%s/%s/Introcourse/%s/%s",
		gitlabutil.IPraktikumGroupName, semesterTag, tutor.GitlabUsername.String, revieweeProfile.GitlabUsername)

	project, _, err := git.Projects.GetProject(projectPath, nil)
	if err != nil {
		if !gitlabutil.IsNotFoundError(err) {
			return fmt.Errorf("find project %q: %w", projectPath, err)
		}

		// Fallback: the project name/path might differ from the GitLab username
		// (e.g. tutor display name vs path). Search the tutor subgroup for a
		// project where the reviewee is a Developer member.
		log.WithField("conventionPath", projectPath).Debug("Convention path not found, falling back to subgroup search")

		tutorSubgroupPath := fmt.Sprintf("ase/%s/%s/Introcourse/%s",
			gitlabutil.IPraktikumGroupName, semesterTag, tutor.GitlabUsername.String)
		tutorSubgroup, _, grpErr := git.Groups.GetGroup(tutorSubgroupPath, nil)
		if grpErr != nil {
			return fmt.Errorf("project %q not found and tutor subgroup %q also not found: %w", projectPath, tutorSubgroupPath, grpErr)
		}

		projects, _, listErr := git.Groups.ListGroupProjects(tutorSubgroup.ID, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
		})
		if listErr != nil {
			return fmt.Errorf("list projects in tutor subgroup %q: %w", tutorSubgroupPath, listErr)
		}

		var found *gitlab.Project
		for _, p := range projects {
			members, _, mErr := git.ProjectMembers.ListProjectMembers(p.ID, &gitlab.ListProjectMembersOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100},
			})
			if mErr != nil {
				continue
			}
			for _, m := range members {
				if m.ID == revieweeGitlabUser.ID && m.AccessLevel >= gitlab.DeveloperPermissions {
					found = p
					break
				}
			}
			if found != nil {
				break
			}
		}
		if found == nil {
			return fmt.Errorf("project %q not found and no project in tutor subgroup %q has reviewee as Developer member", projectPath, tutorSubgroupPath)
		}
		project = found
	}

	// 6. Add reviewer as Reporter (idempotent)
	_, _, err = git.ProjectMembers.AddProjectMember(project.ID, &gitlab.AddProjectMemberOptions{
		UserID:      gitlab.Ptr(reviewerGitlabUser.ID),
		AccessLevel: gitlab.Ptr(gitlab.ReporterPermissions),
	})
	if err != nil && !gitlabutil.IsAlreadyExistsError(err) {
		return fmt.Errorf("add reviewer as Reporter: %w", err)
	}

	// 7. Ensure "Peer Review" approval rule
	err = ensurePeerReviewRule(git, project.ID, reviewerGitlabUser.ID)
	if err != nil {
		return fmt.Errorf("ensure peer review rule: %w", err)
	}

	return nil
}

// getCachedUser resolves a GitLab user by username, using a cache to avoid
// redundant API calls during a single sync run.
func getCachedUser(git *gitlab.Client, username string, cache map[string]*gitlab.User) (*gitlab.User, error) {
	if u, ok := cache[username]; ok {
		return u, nil
	}
	u, err := gitlabutil.GetUser(git, username)
	if err != nil {
		return nil, err
	}
	cache[username] = u
	return u, nil
}

// UnsyncPeerAssignmentsFromGitlab revokes Reporter access and removes "Peer Review"
// approval rules for all current peer assignments. This is the inverse of
// SyncPeerAssignmentsToGitlab — call it before clearing/regenerating assignments.
func UnsyncPeerAssignmentsFromGitlab(ctx context.Context, coursePhaseID uuid.UUID, semesterTag string) ([]peerAssignmentDTO.SyncResult, error) {
	svc := PeerAssignmentServiceSingleton
	if svc.gitlabClient == nil {
		return nil, gitlabutil.ErrClientNotInitialized
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	dbCtx, dbCancel := db.GetTimeoutContext(ctx)
	defer dbCancel()
	assignments, err := svc.queries.GetPeerAssignments(dbCtx, coursePhaseID)
	if err != nil {
		return nil, fmt.Errorf("get peer assignments: %w", err)
	}

	userCache := make(map[string]*gitlab.User)
	var results []peerAssignmentDTO.SyncResult

	for _, assignment := range assignments {
		result := peerAssignmentDTO.SyncResult{
			StudentID: assignment.StudentID,
			PeerID:    assignment.PeerID,
		}

		err := unsyncSinglePeerAccess(ctx, svc, coursePhaseID, assignment.StudentID, assignment.PeerID, semesterTag, userCache)
		if err != nil {
			log.WithFields(log.Fields{
				"studentID": assignment.StudentID,
				"peerID":    assignment.PeerID,
				"error":     err,
			}).Warn("Failed to unsync peer access from GitLab")
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	return results, nil
}

// unsyncSinglePeerAccess revokes a reviewer's Reporter access from the reviewee's
// project and removes them from the "Peer Review" approval rule. Order matters:
// remove from approval rule first, then revoke membership.
func unsyncSinglePeerAccess(ctx context.Context, svc *PeerAssignmentService, coursePhaseID uuid.UUID, reviewerID, revieweeID uuid.UUID, semesterTag string, userCache map[string]*gitlab.User) error {
	git := svc.gitlabClient

	// 1. Look up reviewer's GitLab username
	reviewerProfile, err := svc.queries.GetDeveloperProfileByCourseParticipationID(ctx, db.GetDeveloperProfileByCourseParticipationIDParams{
		CourseParticipationID: reviewerID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get reviewer profile: %w", err)
	}
	if reviewerProfile.GitlabUsername == "" {
		return fmt.Errorf("reviewer %s has no GitLab username", reviewerID)
	}

	// 2. Look up reviewee's GitLab username and their tutor
	revieweeProfile, err := svc.queries.GetDeveloperProfileByCourseParticipationID(ctx, db.GetDeveloperProfileByCourseParticipationIDParams{
		CourseParticipationID: revieweeID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get reviewee profile: %w", err)
	}
	if revieweeProfile.GitlabUsername == "" {
		return fmt.Errorf("reviewee %s has no GitLab username", revieweeID)
	}

	// 3. Get reviewee's assigned tutor for path resolution
	tutor, err := svc.queries.GetAssignedTutor(ctx, db.GetAssignedTutorParams{
		AssignedStudent: pgtype.UUID{Bytes: revieweeID, Valid: true},
		CoursePhaseID:   coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get tutor for reviewee: %w", err)
	}
	if !tutor.GitlabUsername.Valid || tutor.GitlabUsername.String == "" {
		return fmt.Errorf("tutor for reviewee %s has no GitLab username", revieweeID)
	}

	// 4. Resolve reviewer GitLab user ID
	reviewerGitlabUser, err := getCachedUser(git, reviewerProfile.GitlabUsername, userCache)
	if err != nil {
		return fmt.Errorf("resolve reviewer GitLab user: %w", err)
	}

	// 5. Find reviewee's project
	projectPath := fmt.Sprintf("ase/%s/%s/Introcourse/%s/%s",
		gitlabutil.IPraktikumGroupName, semesterTag, tutor.GitlabUsername.String, revieweeProfile.GitlabUsername)

	project, _, err := git.Projects.GetProject(projectPath, nil)
	if err != nil {
		if gitlabutil.IsNotFoundError(err) {
			return nil // project doesn't exist — nothing to revoke
		}
		return fmt.Errorf("find project %q: %w", projectPath, err)
	}

	// 6. Remove reviewer from "Peer Review" approval rule (do this BEFORE revoking membership)
	if err := removePeerFromReviewRule(git, project.ID, reviewerGitlabUser.ID); err != nil {
		return fmt.Errorf("remove from peer review rule: %w", err)
	}

	// 7. Revoke Reporter access (idempotent — 404 means already removed)
	_, err = git.ProjectMembers.DeleteProjectMember(project.ID, reviewerGitlabUser.ID)
	if err != nil && !gitlabutil.IsNotFoundError(err) {
		return fmt.Errorf("revoke Reporter access: %w", err)
	}

	return nil
}

// removePeerFromReviewRule removes a user from the "Peer Review" approval rule.
// If the user was the last one on the rule, the rule is deleted entirely.
// If the rule or user is not found, this is a no-op.
func removePeerFromReviewRule(git *gitlab.Client, projectID int64, userID int64) error {
	rules, _, err := git.Projects.GetProjectApprovalRules(projectID, &gitlab.GetProjectApprovalRulesListsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	})
	if err != nil {
		return fmt.Errorf("list approval rules: %w", err)
	}

	for _, r := range rules {
		if r.Name != peerReviewRuleName {
			continue
		}

		// Build remaining user list (excluding the user to remove)
		var remaining []int64
		found := false
		for _, u := range r.Users {
			if u.ID == userID {
				found = true
			} else {
				remaining = append(remaining, u.ID)
			}
		}

		if !found {
			return nil // user not in this rule — nothing to do
		}

		if len(remaining) == 0 {
			// Last user — delete the entire rule
			_, err := git.Projects.DeleteProjectApprovalRule(projectID, r.ID)
			if err != nil && !gitlabutil.IsNotFoundError(err) {
				return fmt.Errorf("delete approval rule: %w", err)
			}
		} else {
			// Update rule with remaining users
			_, _, err := git.Projects.UpdateProjectApprovalRule(projectID, r.ID, &gitlab.UpdateProjectLevelRuleOptions{
				UserIDs: gitlab.Ptr(remaining),
			})
			if err != nil {
				return fmt.Errorf("update approval rule: %w", err)
			}
		}
		return nil
	}

	return nil // rule not found — nothing to do
}

// ensurePeerReviewRule creates or updates the "Peer Review" approval rule on the
// project. If the rule already exists, it updates the user list to include the
// new reviewer. Uses r.Users (explicitly assigned) rather than r.EligibleApprovers
// (which includes inherited group members).
func ensurePeerReviewRule(git *gitlab.Client, projectID int64, reviewerGitlabUserID int64) error {
	rules, _, err := git.Projects.GetProjectApprovalRules(projectID, &gitlab.GetProjectApprovalRulesListsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	})
	if err != nil {
		return fmt.Errorf("list approval rules: %w", err)
	}

	for _, r := range rules {
		if r.Name == peerReviewRuleName {
			// Check if reviewer is already in the rule (use Users, not EligibleApprovers)
			for _, u := range r.Users {
				if u.ID == reviewerGitlabUserID {
					return nil // already configured
				}
			}
			// Add reviewer to existing rule — only send UserIDs to avoid clobbering other fields
			existingUserIDs := make([]int64, 0, len(r.Users)+1)
			for _, u := range r.Users {
				existingUserIDs = append(existingUserIDs, u.ID)
			}
			existingUserIDs = append(existingUserIDs, reviewerGitlabUserID)
			_, _, updateErr := git.Projects.UpdateProjectApprovalRule(projectID, r.ID, &gitlab.UpdateProjectLevelRuleOptions{
				UserIDs: gitlab.Ptr(existingUserIDs),
			})
			if updateErr != nil {
				return fmt.Errorf("update approval rule: %w", updateErr)
			}
			return nil
		}
	}

	// Create new rule — ApprovalsRequired=0 makes this non-blocking so students
	// can merge with only tutor approval. Peers still appear as eligible approvers.
	_, _, err = git.Projects.CreateProjectApprovalRule(projectID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr(peerReviewRuleName),
		ApprovalsRequired: gitlab.Ptr(int64(0)),
		UserIDs:           gitlab.Ptr([]int64{reviewerGitlabUserID}),
	})
	if err != nil {
		return fmt.Errorf("create approval rule: %w", err)
	}
	return nil
}
