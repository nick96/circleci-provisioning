package main

import "testing"

func TestFmtUri(t *testing.T) {
	type args struct { resource string; action string }
	type test struct { name string; input args; project *CircleCIProject; expected string }

	testCases := []test{
		{
			name: "Basic",
			input: args{ "project", "follow" },
			project: NewCircleCIProject("git", "test", "test", "token"),
			expected: "https://circleci.com/api/v1.1/project/git/test/test/follow?circle-token=token",
		},
		{
			name: "Project name with spaces",
			input: args{ "resource", "action" },
			project: NewCircleCIProject("git", "owner", "project name", "token"),
			expected: "https://circleci.com/api/v1.1/resource/git/owner/project%20name/action?circle-token=token",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func (t *testing.T) {
			t.Parallel()
			actual := tc.project.fmtURI(tc.input.resource, tc.input.action)
			if actual != tc.expected {
				t.Errorf("Expected %s found %s", tc.expected, actual)
			}
		})
	}
}
