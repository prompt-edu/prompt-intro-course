package infrastructureSetup

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/prompt-edu/prompt-intro-course/server/db/sqlc"
	"github.com/prompt-edu/prompt-intro-course/server/infrastructureSetup/infrastructureDTO"
	log "github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type InfrastructureService struct {
	queries      db.Queries
	conn         *pgxpool.Pool
	gitlabClient *gitlab.Client
}

var InfrastructureServiceSingleton *InfrastructureService

const TOP_LEVEL_GROUP_NAME = "ASE"
const I_PRAKTIKUM_GROUP_NAME = "iPraktikum"

func CreateCourseInfrastructure(coursePhaseID uuid.UUID, semesterTag string) error {
	// 1.) Get Top Level Group
	ipraktikumGroup, err := getiPraktikumGroup()
	if err != nil {
		return err
	}

	courseGroup, err := createCourseIterationGroup(semesterTag, ipraktikumGroup.ID)
	if err != nil {
		return err
	}

	// Steps 2-5 are independent — collect all errors instead of failing fast
	var errs []error

	if _, err = createDeveloperTopLevelGroup(courseGroup.ID); err != nil {
		errs = append(errs, fmt.Errorf("create developer group: %w", err))
	}

	if _, err = createTeachingGroup(courseGroup.ID, "tutors"); err != nil {
		errs = append(errs, fmt.Errorf("create tutors group: %w", err))
	}

	if _, err = createTeachingGroup(courseGroup.ID, "coaches"); err != nil {
		errs = append(errs, fmt.Errorf("create coaches group: %w", err))
	}

	if _, err = createTeachingGroup(courseGroup.ID, "Introcourse"); err != nil {
		errs = append(errs, fmt.Errorf("create Introcourse group: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func CreateStudentInfrastructure(ctx context.Context, coursePhaseID, courseParticipationID uuid.UUID, semesterTag, repoName, studentName, submissionDeadline string) error {
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

	log.Info("Creating student repo for student: ", devProfile.GitlabUsername, " with tutor: ", tutor.AssignedTutor)

	// 3.) Get Gitlab IDs
	studentGitlabUser, err := getUserID(devProfile.GitlabUsername)
	if err != nil {
		return fmt.Errorf("get student GitLab ID for %q: %w", devProfile.GitlabUsername, err)
	}

	tutorGitlabUser, err := getUserID(tutor.GitlabUsername.String)
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

	// 5.) Create the student project (fully idempotent — safe to re-run)
	err = CreateStudentProject(repoName, studentGitlabUser.ID, tutorGitlabUser.ID, introCourseGroup.ID, introCourseGroup.FullPath, developerGroup.ID, studentName, submissionDeadline)
	if err != nil {
		log.WithField("student", repoName).Error("Failed to create student project: ", err)
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
	ipraktikumGroup, err := getSubGroup(I_PRAKTIKUM_GROUP_NAME, ASE_GROUP_ID)
	if err != nil {
		return nil, fmt.Errorf("get iPraktikum group: %w", err)
	}

	return ipraktikumGroup, nil
}

func GetAllStudentGitlabStatus(c context.Context, coursePhaseID uuid.UUID) ([]infrastructureDTO.GitlabStatus, error) {
	ctxWithTimeout, cancel := db.GetTimeoutContext(c)
	defer cancel()

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
