package infrastructureSetup

import (
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const courseBoardName = "Course work"

var courseBoardStatuses = []string{"Open", "In Progress", "In Review", "Blocked", "Done"}

type statusBoardQuery struct {
	Data struct {
		Project *struct {
			Group struct {
				Lifecycles struct {
					Nodes []struct {
						Name     string `json:"name"`
						Statuses []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"statuses"`
					} `json:"nodes"`
				} `json:"lifecycles"`
			} `json:"group"`
			Boards struct {
				Nodes []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Lists struct {
						Nodes []struct {
							ID       string `json:"id"`
							Position int    `json:"position"`
							Status   *struct {
								ID string `json:"id"`
							} `json:"status"`
						} `json:"nodes"`
					} `json:"lists"`
				} `json:"nodes"`
			} `json:"boards"`
		} `json:"project"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func readStatusBoard(git *gitlab.Client, projectPath string) (*statusBoardQuery, error) {
	var result statusBoardQuery
	_, err := git.GraphQL.Do(gitlab.GraphQLQuery{
		Query:     `query($path: ID!) { project(fullPath: $path) { group { lifecycles { nodes { name statuses { id name } } } } boards(first: 20) { nodes { id name lists(first: 20) { nodes { id position status { id } } } } } } }`,
		Variables: map[string]any{"path": projectPath},
	}, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Errors) > 0 || result.Data.Project == nil {
		return nil, fmt.Errorf("GitLab could not read the project status lifecycle: %v", result.Errors)
	}
	return &result, nil
}

func requiredStatusIDs(result *statusBoardQuery) (map[string]string, error) {
	ids := make(map[string]string)
	for _, lifecycle := range result.Data.Project.Group.Lifecycles.Nodes {
		if lifecycle.Name != "Default" {
			continue
		}
		for _, status := range lifecycle.Statuses {
			ids[status.Name] = status.ID
		}
	}
	for _, name := range courseBoardStatuses {
		if ids[name] == "" {
			return nil, fmt.Errorf("GitLab lifecycle is missing status %q", name)
		}
	}
	return ids, nil
}

func ensureCourseStatusBoard(git *gitlab.Client, projectID int64) error {
	project, _, err := git.Projects.GetProject(projectID, nil)
	if err != nil {
		return err
	}
	state, err := readStatusBoard(git, project.PathWithNamespace)
	if err != nil {
		return err
	}
	ids, err := requiredStatusIDs(state)
	if err != nil {
		return err
	}
	boards, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.IssueBoard, *gitlab.Response, error) {
		return git.Boards.ListIssueBoards(projectID, &gitlab.ListIssueBoardsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return err
	}
	var board *gitlab.IssueBoard
	for _, candidate := range boards {
		if candidate.Name == courseBoardName {
			if board != nil {
				return fmt.Errorf("multiple %q boards exist", courseBoardName)
			}
			board = candidate
		}
	}
	if board == nil {
		board, _, err = git.Boards.CreateIssueBoard(projectID, &gitlab.CreateIssueBoardOptions{Name: gitlab.Ptr(courseBoardName)})
		if err != nil {
			return err
		}
	}
	if !board.HideBacklogList || !board.HideClosedList {
		board, _, err = git.Boards.UpdateIssueBoard(projectID, board.ID, &gitlab.UpdateIssueBoardOptions{
			HideBacklogList: gitlab.Ptr(true), HideClosedList: gitlab.Ptr(true),
		})
		if err != nil {
			return err
		}
	}
	if !board.HideBacklogList || !board.HideClosedList {
		return fmt.Errorf("GitLab did not hide the redundant Open/Closed board columns")
	}
	for position, name := range courseBoardStatuses {
		state, err = readStatusBoard(git, project.PathWithNamespace)
		if err != nil {
			return err
		}
		boardID := fmt.Sprintf("gid://gitlab/Board/%d", board.ID)
		found := false
		for _, candidate := range state.Data.Project.Boards.Nodes {
			if candidate.ID != boardID {
				continue
			}
			for _, list := range candidate.Lists.Nodes {
				if list.Status != nil && list.Status.ID == ids[name] {
					found = true
				}
			}
		}
		if found {
			continue
		}
		var created struct {
			Data struct {
				BoardListCreate struct {
					Errors []string `json:"errors"`
				} `json:"boardListCreate"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		_, err = git.GraphQL.Do(gitlab.GraphQLQuery{
			Query:     `mutation($board: BoardID!, $status: WorkItemsStatusesStatusID!, $position: Int) { boardListCreate(input: {boardId: $board, statusId: $status, position: $position}) { errors } }`,
			Variables: map[string]any{"board": boardID, "status": ids[name], "position": position},
		}, &created)
		if err != nil {
			return err
		}
		if len(created.Errors) > 0 || len(created.Data.BoardListCreate.Errors) > 0 {
			return fmt.Errorf("create %q status column: %v %s", name, created.Errors, strings.Join(created.Data.BoardListCreate.Errors, ", "))
		}
	}
	return courseStatusBoardMatches(git, project.PathWithNamespace, board.ID, ids)
}

func courseStatusBoardMatches(git *gitlab.Client, projectPath string, boardID int64, ids map[string]string) error {
	state, err := readStatusBoard(git, projectPath)
	if err != nil {
		return err
	}
	wantedBoard := fmt.Sprintf("gid://gitlab/Board/%d", boardID)
	for _, board := range state.Data.Project.Boards.Nodes {
		if board.ID != wantedBoard {
			continue
		}
		positions := make(map[string]int)
		for _, list := range board.Lists.Nodes {
			if list.Status != nil {
				positions[list.Status.ID] = list.Position
			}
		}
		if len(positions) != len(courseBoardStatuses) {
			return fmt.Errorf("course board has unexpected status columns")
		}
		for position, name := range courseBoardStatuses {
			if positions[ids[name]] != position {
				return fmt.Errorf("course board status columns are out of order")
			}
		}
		return nil
	}
	return fmt.Errorf("course board is missing")
}

func checkCourseStatusBoard(git *gitlab.Client, projectID int64, projectPath string) error {
	state, err := readStatusBoard(git, projectPath)
	if err != nil {
		return err
	}
	ids, err := requiredStatusIDs(state)
	if err != nil {
		return err
	}
	boards, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.IssueBoard, *gitlab.Response, error) {
		return git.Boards.ListIssueBoards(projectID, &gitlab.ListIssueBoardsOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}, p)
	})
	if err != nil {
		return err
	}
	for _, board := range boards {
		if board.Name == courseBoardName && board.HideBacklogList && board.HideClosedList {
			return courseStatusBoardMatches(git, projectPath, board.ID, ids)
		}
	}
	return fmt.Errorf("course status board is missing or not configured")
}

// A practiced demo may still have unchanged issue text but changed workflow
// status. It is not a clean instructor copy until every work item is Open.
func demoWorkItemsClean(git *gitlab.Client, projectPath string, expectedCount int) (bool, error) {
	state, err := readStatusBoard(git, projectPath)
	if err != nil {
		return false, err
	}
	ids, err := requiredStatusIDs(state)
	if err != nil {
		return false, err
	}
	var result struct {
		Data struct {
			Project struct {
				WorkItems struct {
					Nodes []struct {
						Widgets []struct {
							Status *struct {
								ID string `json:"id"`
							} `json:"status"`
						} `json:"widgets"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"workItems"`
			} `json:"project"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	_, err = git.GraphQL.Do(gitlab.GraphQLQuery{
		Query:     `query($path: ID!) { project(fullPath: $path) { workItems(first: 100) { nodes { widgets { ... on WorkItemWidgetStatus { status { id } } } } pageInfo { hasNextPage } } } }`,
		Variables: map[string]any{"path": projectPath},
	}, &result)
	if err != nil {
		return false, err
	}
	if len(result.Errors) > 0 {
		return false, fmt.Errorf("read demo work-item statuses: %v", result.Errors)
	}
	items := result.Data.Project.WorkItems
	if items.PageInfo.HasNextPage || len(items.Nodes) != expectedCount {
		return false, nil
	}
	for _, item := range items.Nodes {
		open := false
		for _, widget := range item.Widgets {
			if widget.Status != nil && widget.Status.ID == ids["Open"] {
				open = true
			}
		}
		if !open {
			return false, nil
		}
	}
	return true, nil
}
