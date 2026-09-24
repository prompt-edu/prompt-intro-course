package infrastructureSetup

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/gitlabutil"
	"github.com/prompt-edu/prompt-intro-course/server/infrastructureSetup/infrastructureDTO"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type InfrastructureService struct {
	queries                   db.Queries
	conn                      *pgxpool.Pool
	gitlabClient              *gitlab.Client
	teachingMaterialProjectID string
	templates                 templateCache
	issues                    issueCache
	cicd                      cicdCache
}

var InfrastructureServiceSingleton *InfrastructureService

var iPraktikumGroupName = gitlabutil.IPraktikumGroupName

func CreateCourseInfrastructure(ctx context.Context, coursePhaseID uuid.UUID, semesterTag string) error {
	git, err := getClient()
	if err != nil {
		return err
	}
	material, err := loadMaterialSnapshot(git, InfrastructureServiceSingleton.teachingMaterialProjectID, "")
	if err != nil {
		return fmt.Errorf("load current teaching material: %w", err)
	}
	// 1.) Get Top Level Group
	ipraktikumGroup, err := getiPraktikumGroup()
	if err != nil {
		return err
	}

	courseGroup, err := createCourseIterationGroup(semesterTag, ipraktikumGroup.ID)
	if err != nil {
		return err
	}

	// Steps 2-4 are independent — collect all errors instead of failing fast
	var errs []error

	// 2.) Create the developer group
	if _, err = createDeveloperTopLevelGroup(courseGroup.ID); err != nil {
		errs = append(errs, fmt.Errorf("create developer group: %w", err))
	}

	// 3.) Create the tutor groups
	tutorsGroup, tutorsErr := createTeachingGroup(courseGroup.ID, "tutors")
	if tutorsErr != nil {
		errs = append(errs, fmt.Errorf("create tutors group: %w", tutorsErr))
	}

	// 4.) Create the coach group
	if _, err = createTeachingGroup(courseGroup.ID, "coaches"); err != nil {
		errs = append(errs, fmt.Errorf("create coaches group: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// 5.) Create the introCourse group (fail-fast: demo project depends on it)
	introCourseGroup, err := createTeachingGroup(courseGroup.ID, "Introcourse")
	if err != nil {
		return fmt.Errorf("create Introcourse group: %w", err)
	}

	// Tutors need access to the demo and every future student subgroup, not only
	// their own subgroup. Share the parent once so projects inherit that access.
	if err = ensureTutorGroupAccess(git, introCourseGroup.ID, tutorsGroup.ID); err != nil {
		return fmt.Errorf("share Introcourse group with tutors: %w", err)
	}
	if err = ensureImportedTutorGroupMembers(ctx, coursePhaseID, git, tutorsGroup.ID); err != nil {
		return fmt.Errorf("sync imported tutors to GitLab: %w", err)
	}

	// 6.) The shared CI config is required by every student project. Do not mark
	// course setup complete if it is missing or could not be updated.
	if err = createCICDProjectWithMaterial(git, introCourseGroup.ID, introCourseGroup.FullPath, material); err != nil {
		return fmt.Errorf("set up shared CI/CD project: %w", err)
	}

	// 7.) The demo is the course's reference project and setup smoke test.
	if err = createDemoProjectWithMaterial(git, introCourseGroup.ID, introCourseGroup.FullPath, tutorsGroup.ID, material); err != nil {
		return fmt.Errorf("set up demo project: %w", err)
	}

	return nil
}

func ensureImportedTutorGroupMembers(ctx context.Context, coursePhaseID uuid.UUID, git *gitlab.Client, groupID int64) error {
	tutors, err := InfrastructureServiceSingleton.queries.GetAllTutors(ctx, coursePhaseID)
	if err != nil {
		return fmt.Errorf("get imported tutors: %w", err)
	}
	if len(tutors) == 0 {
		return fmt.Errorf("import tutors before setting up course repositories")
	}
	var unresolved int
	for _, tutor := range tutors {
		if !tutor.GitlabUsername.Valid || tutor.GitlabUsername.String == "" {
			unresolved++
			continue
		}
		user, lookupErr := gitlabutil.GetUser(git, tutor.GitlabUsername.String)
		if lookupErr != nil {
			unresolved++
			continue
		}
		if memberErr := ensureGroupMember(git, groupID, user.ID, gitlab.DeveloperPermissions); memberErr != nil {
			return fmt.Errorf("add imported tutor to GitLab group: %w", memberErr)
		}
	}
	if unresolved > 0 {
		return fmt.Errorf("%d imported tutor GitLab username(s) are missing or unresolved", unresolved)
	}
	return nil
}

func CreateStudentInfrastructure(ctx context.Context, coursePhaseID, courseParticipationID uuid.UUID, semesterTag, repoName, studentName, submissionDeadline string) error {
	status, err := CourseInfrastructureStatus(ctx, coursePhaseID, semesterTag)
	if err != nil {
		return fmt.Errorf("check demo readiness: %w", err)
	}
	if !status.Checks.DemoReady {
		return fmt.Errorf("demo repository is not ready for current teaching material; repair and test it before creating student repos")
	}
	git, err := getClient()
	if err != nil {
		return err
	}
	material, err := loadMaterialSnapshot(git, InfrastructureServiceSingleton.teachingMaterialProjectID, status.Source.SHA)
	if err != nil {
		return fmt.Errorf("load verified teaching material: %w", err)
	}
	// 1.) get the student developer profile
	devProfile, err := InfrastructureServiceSingleton.queries.GetDeveloperProfileByCourseParticipationID(ctx, db.GetDeveloperProfileByCourseParticipationIDParams{
		CourseParticipationID: courseParticipationID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get developer profile: %w", err)
	}
	if devProfile.GitlabUsername == "" {
		return fmt.Errorf("cannot create student repo: missing GitLab username for participation %s", courseParticipationID)
	}

	// 2.) Get the assigned tutor
	tutor, err := InfrastructureServiceSingleton.queries.GetAssignedTutor(ctx, db.GetAssignedTutorParams{
		AssignedStudent: pgtype.UUID{Bytes: courseParticipationID, Valid: true},
		CoursePhaseID:   coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("get assigned tutor: %w", err)
	}
	if !tutor.GitlabUsername.Valid || tutor.GitlabUsername.String == "" {
		return fmt.Errorf("cannot create student repo: missing tutor GitLab username for participation %s", courseParticipationID)
	}
	if !tutor.AssignedTutor.Valid {
		return fmt.Errorf("cannot create student repo: no tutor assigned for participation %s", courseParticipationID)
	}

	log.WithFields(log.Fields{
		"student": devProfile.GitlabUsername,
		"tutor":   tutor.GitlabUsername.String,
	}).Info("Creating student repo")

	// 3.) Get Gitlab IDs
	studentGitlabUser, err := getUser(devProfile.GitlabUsername)
	if err != nil {
		return fmt.Errorf("get student GitLab ID for %q: %w", devProfile.GitlabUsername, err)
	}

	tutorGitlabUser, err := getUser(tutor.GitlabUsername.String)
	if err != nil {
		return fmt.Errorf("get tutor GitLab ID for %q: %w", tutor.GitlabUsername.String, err)
	}

	// 4.) Get required GitLab groups
	ipraktikumGroup, err := getiPraktikumGroup()
	if err != nil {
		return err
	}

	semesterGroup, err := getSubGroup(semesterTag, ipraktikumGroup.ID)
	if err != nil {
		return fmt.Errorf("get semester group %q: %w", semesterTag, err)
	}

	introCourseGroup, err := getSubGroup("Introcourse", semesterGroup.ID)
	if err != nil {
		return fmt.Errorf("get Introcourse group: %w", err)
	}

	developerGroup, err := getSubGroup("developer", semesterGroup.ID)
	if err != nil {
		return fmt.Errorf("get developer group: %w", err)
	}

	tutorsGroup, err := getSubGroup("tutors", semesterGroup.ID)
	if err != nil {
		return fmt.Errorf("get tutors group: %w", err)
	}

	// 5.) Get or create tutor subgroup inside Introcourse
	tutorSubgroupID, tutorSubgroupPath, err := getOrCreateTutorSubgroup(
		tutor.GitlabUsername.String, tutor.FirstName, tutor.LastName,
		tutorGitlabUser.ID, introCourseGroup.ID,
	)
	if err != nil {
		return fmt.Errorf("get/create tutor subgroup: %w", err)
	}

	// 6.) Create the student project in tutor's subgroup (fully idempotent)
	err = createStudentProjectWithMaterial(StudentProjectParams{
		RepoName:             repoName,
		DevID:                studentGitlabUser.ID,
		TutorSubgroupID:      tutorSubgroupID,
		TutorSubgroupPath:    tutorSubgroupPath,
		TutorsGroupID:        tutorsGroup.ID,
		DevGroupID:           developerGroup.ID,
		IntroCourseGroupPath: introCourseGroup.FullPath,
		StudentName:          studentName,
		SubmissionDeadline:   submissionDeadline,
	}, material)
	if err != nil {
		log.WithField("student", repoName).Error("Failed to create student project: ", err)
		// store error in the db
		dbError := InfrastructureServiceSingleton.queries.AddGitlabError(ctx, db.AddGitlabErrorParams{
			CourseParticipationID: courseParticipationID,
			CoursePhaseID:         coursePhaseID,
			ErrorMessage:          pgtype.Text{String: err.Error(), Valid: true},
		})
		if dbError != nil {
			log.Error("Failed to store gitlab error in db: ", dbError)
		}
		return err
	}

	err = InfrastructureServiceSingleton.queries.AddGitlabStatus(ctx, db.AddGitlabStatusParams{
		CourseParticipationID: courseParticipationID,
		CoursePhaseID:         coursePhaseID,
	})

	if err != nil {
		return fmt.Errorf("update gitlab status in db: %w", err)
	}

	return nil
}

func getiPraktikumGroup() (*gitlab.Group, error) {
	ipraktikumGroup, err := getSubGroup(iPraktikumGroupName, aseGroupID)
	if err != nil {
		return nil, fmt.Errorf("get iPraktikum group: %w", err)
	}

	return ipraktikumGroup, nil

}

func GetAllStudentGitlabStatus(c context.Context, coursePhaseID uuid.UUID) ([]infrastructureDTO.GitlabStatus, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(c)
	defer cancel()

	// 1.) Get all gitlab status
	gitlabStatuses, err := InfrastructureServiceSingleton.queries.GetAllGitlabStatus(ctxWithTimeout, coursePhaseID)
	if err != nil {
		return nil, fmt.Errorf("get gitlab statuses: %w", err)
	}

	return infrastructureDTO.GetGitlabStatusDTOsFromModels(gitlabStatuses), nil

}

func ManuallyOverwriteStudentGitlabStatus(c context.Context, coursePhaseID, courseParticipationID uuid.UUID) error {
	ctxWithTimeout, cancel := db.GetTimeoutContext(c)
	defer cancel()

	err := InfrastructureServiceSingleton.queries.AddGitlabStatus(ctxWithTimeout, db.AddGitlabStatusParams{
		CourseParticipationID: courseParticipationID,
		CoursePhaseID:         coursePhaseID,
	})
	if err != nil {
		return fmt.Errorf("update gitlab status in db: %w", err)
	}
	return nil
}
