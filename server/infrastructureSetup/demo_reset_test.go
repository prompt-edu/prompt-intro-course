package infrastructureSetup

import (
	"testing"

	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestDemoTemplateActionsReplaceOnlyChanges(t *testing.T) {
	actions, err := demoTemplateActions(map[string]demoRootFile{
		"README.md":            {content: "For Demo\n"},
		".githooks/pre-commit": {content: "old", executable: true},
		"old.txt":              {content: "remove"},
	}, []templateFile{
		{Path: "README.md", Content: "For {{.StudentName}}\n"},
		{Path: ".githooks/pre-commit", Content: "new", ExecuteFilemode: true},
		{Path: "new.txt", Content: "add"},
	})
	require.NoError(t, err)
	require.Len(t, actions, 3)
	byPath := make(map[string]*gitlab.CommitActionOptions)
	for _, action := range actions {
		byPath[*action.FilePath] = action
	}
	require.NotContains(t, byPath, "README.md")
	require.Equal(t, gitlab.FileUpdate, *byPath[".githooks/pre-commit"].Action)
	require.True(t, *byPath[".githooks/pre-commit"].ExecuteFilemode)
	require.Equal(t, gitlab.FileCreate, *byPath["new.txt"].Action)
	require.Equal(t, gitlab.FileDelete, *byPath["old.txt"].Action)
}

func TestDemoTemplateActionsRejectDuplicatePaths(t *testing.T) {
	_, err := demoTemplateActions(nil, []templateFile{{Path: "README.md"}, {Path: "README.md"}})
	require.ErrorContains(t, err, "duplicate")
}

func TestDemoTemplateActionsAllowsUnchangedTemplate(t *testing.T) {
	actions, err := demoTemplateActions(map[string]demoRootFile{
		"README.md": {content: "For Demo\n"},
	}, []templateFile{{Path: "README.md", Content: "For {{.StudentName}}\n"}})
	require.NoError(t, err)
	require.Empty(t, actions)
}
