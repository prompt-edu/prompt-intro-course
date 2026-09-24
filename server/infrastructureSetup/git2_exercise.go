package infrastructureSetup

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// The two exercise branches share the student's repository but never change
// its main branch. The fixture is sourced from the same immutable teaching
// material revision as the starter files and daily issues.
var git2ExerciseBranches = []struct {
	name  string
	dir   string
	files []string
}{
	{"exercise/git2-navigation", "navigation", []string{"ContentView.swift", "BirdListView.swift"}},
	{"exercise/git2-rows", "rows", []string{"ContentView.swift", "BirdRow.swift"}},
}

func fetchGit2ExerciseAtRef(git *gitlab.Client, materialProjectID, ref string) (map[string][]templateFile, error) {
	result := make(map[string][]templateFile, len(git2ExerciseBranches))
	for _, branch := range git2ExerciseBranches {
		for _, name := range branch.files {
			sourcePath := "git2_exercise/" + branch.dir + "/" + name
			raw, _, err := git.RepositoryFiles.GetRawFile(materialProjectID, sourcePath, &gitlab.GetRawFileOptions{
				Ref: gitlab.Ptr(ref),
			})
			if err != nil {
				return nil, fmt.Errorf("read Git 2 exercise file %q: %w", sourcePath, err)
			}
			if len(raw) == 0 || len(raw) > maxTemplateBytes {
				return nil, fmt.Errorf("git 2 exercise file %q has invalid size %d", sourcePath, len(raw))
			}
			result[branch.name] = append(result[branch.name], templateFile{
				Path:    "IntrocourseApp/" + name,
				Content: string(raw),
			})
		}
	}
	return result, nil
}

func ensureGit2ExerciseBranches(git *gitlab.Client, projectID int64, repoName string, exercise map[string][]templateFile) error {
	for _, branch := range git2ExerciseBranches {
		_, _, err := git.Branches.GetBranch(projectID, branch.name)
		if err == nil {
			// A retry must not overwrite work on an existing exercise branch.
			if err := protectGit2ExerciseBranch(git, projectID, branch.name); err != nil {
				return fmt.Errorf("protect Git 2 branch %q in %q: %w", branch.name, repoName, err)
			}
			continue
		}
		if !isNotFoundError(err) {
			return fmt.Errorf("check Git 2 branch %q in %q: %w", branch.name, repoName, err)
		}
		files := exercise[branch.name]
		if len(files) != len(branch.files) {
			return fmt.Errorf("git 2 branch %q has incomplete teaching material", branch.name)
		}
		actions := make([]*gitlab.CommitActionOptions, 0, len(files))
		for _, file := range files {
			action := gitlab.FileCreate
			if file.Path == "IntrocourseApp/ContentView.swift" {
				action = gitlab.FileUpdate
			}
			actions = append(actions, &gitlab.CommitActionOptions{
				Action:   gitlab.Ptr(action),
				FilePath: gitlab.Ptr(file.Path),
				Content:  gitlab.Ptr(file.Content),
			})
		}
		_, _, err = git.Commits.CreateCommit(projectID, &gitlab.CreateCommitOptions{
			Branch:        gitlab.Ptr(branch.name),
			StartBranch:   gitlab.Ptr("main"),
			CommitMessage: gitlab.Ptr("Prepare Git 2 merge-conflict exercise"),
			Actions:       actions,
		})
		if err != nil {
			// Concurrent student setup requests can race. An already-created branch
			// is the desired result; any other failure must be reported.
			if _, _, readErr := git.Branches.GetBranch(projectID, branch.name); readErr != nil {
				return fmt.Errorf("create Git 2 branch %q in %q: %w", branch.name, repoName, err)
			}
		}
		if err := protectGit2ExerciseBranch(git, projectID, branch.name); err != nil {
			return fmt.Errorf("protect Git 2 branch %q in %q: %w", branch.name, repoName, err)
		}
	}
	return nil
}

// Keep the shared exercise inputs fixed. Students merge them into a local
// practice branch; neither students nor peers need to push to these branches.
func protectGit2ExerciseBranch(git *gitlab.Client, projectID int64, name string) error {
	protected, _, err := git.ProtectedBranches.GetProtectedBranch(projectID, name)
	if isNotFoundError(err) {
		protected, _, err = git.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
			Name: gitlab.Ptr(name), PushAccessLevel: gitlab.Ptr(gitlab.NoPermissions),
			MergeAccessLevel: gitlab.Ptr(gitlab.NoPermissions), AllowForcePush: gitlab.Ptr(false),
		})
		if isAlreadyExistsError(err) {
			protected, _, err = git.ProtectedBranches.GetProtectedBranch(projectID, name)
		}
	}
	if err != nil {
		return err
	}
	if !git2ExerciseProtectionMatches(protected, name) {
		return fmt.Errorf("GitLab did not confirm no-push, no-merge protection")
	}
	return nil
}

func git2ExerciseProtectionMatches(protected *gitlab.ProtectedBranch, name string) bool {
	return protected != nil && protected.Name == name && !protected.AllowForcePush &&
		len(protected.PushAccessLevels) == 1 && protected.PushAccessLevels[0].AccessLevel == gitlab.NoPermissions &&
		len(protected.MergeAccessLevels) == 1 && protected.MergeAccessLevels[0].AccessLevel == gitlab.NoPermissions
}

func demoGit2ExerciseIsClean(git *gitlab.Client, materialProjectID, sourceSHA string, demoID int64, mainSHA string, branches []*gitlab.Branch) (bool, error) {
	byName := make(map[string]*gitlab.Branch, len(branches))
	for _, branch := range branches {
		byName[branch.Name] = branch
	}
	if len(byName) != len(git2ExerciseBranches)+1 || byName["main"] == nil {
		return false, nil
	}
	fixture, err := fetchGit2ExerciseAtRef(git, materialProjectID, sourceSHA)
	if err != nil {
		return false, err
	}
	for _, spec := range git2ExerciseBranches {
		branch := byName[spec.name]
		if branch == nil || !branch.Protected || branch.Commit == nil ||
			len(branch.Commit.ParentIDs) != 1 || branch.Commit.ParentIDs[0] != mainSHA {
			return false, nil
		}
		protected, _, err := git.ProtectedBranches.GetProtectedBranch(demoID, spec.name)
		if err != nil {
			return false, fmt.Errorf("get demo Git 2 branch protection: %w", err)
		}
		if !git2ExerciseProtectionMatches(protected, spec.name) {
			return false, nil
		}
		diffs, _, err := git.Commits.GetCommitDiff(demoID, branch.Commit.ID, nil)
		if err != nil {
			return false, fmt.Errorf("get demo Git 2 branch diff: %w", err)
		}
		if len(diffs) != len(fixture[spec.name]) {
			return false, nil
		}
		changed := make(map[string]bool, len(diffs))
		for _, diff := range diffs {
			if diff.DeletedFile || diff.RenamedFile {
				return false, nil
			}
			changed[diff.NewPath] = true
		}
		for _, file := range fixture[spec.name] {
			if !changed[file.Path] {
				return false, nil
			}
			actual, _, err := git.RepositoryFiles.GetRawFile(demoID, file.Path, &gitlab.GetRawFileOptions{Ref: gitlab.Ptr(spec.name)})
			if err != nil {
				return false, fmt.Errorf("read demo Git 2 branch file %q: %w", file.Path, err)
			}
			if string(actual) != file.Content {
				return false, nil
			}
		}
	}
	return true, nil
}
