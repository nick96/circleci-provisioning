package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFmtUri(t *testing.T) {
	type args struct {
		resource string
		action   string
	}
	type test struct {
		input    args
		project  *CircleCIProject
		expected string
	}

	testCases := []test{
		{
			input:    args{"project", "follow"},
			project:  NewCircleCIProject("git", "test", "test", "token"),
			expected: "https://circleci.com/api/v1.1/project/git/test/test/follow?circle-token=token",
		},
		{
			input:    args{"resource", "action"},
			project:  NewCircleCIProject("git", "owner", "project name", "token"),
			expected: "https://circleci.com/api/v1.1/resource/git/owner/project%20name/action?circle-token=token",
		},
	}

	for _, tc := range testCases {
		actual := tc.project.fmtURI(tc.input.resource, tc.input.action)
		if actual != tc.expected {
			t.Errorf("Expected %s found %s", tc.expected, actual)
		}
	}
}

func TestFollowHappy(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, "ok")
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, svr.Listener.Addr().String())
			},
		},
	}
	client := &CircleCIClient{"http://localhost", httpClient}

	project := CircleCIProject{"git", "test", "test", "token", client}

	err := project.Follow()
	if err != nil {
		t.Errorf("Expected no error, found: %v", err)
	}
}

func TestFollowUnhappy(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "bad")
	})
	svr := httptest.NewServer(handler)
	defer svr.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, svr.Listener.Addr().String())
			},
		},
	}
	client := &CircleCIClient{"http://localhost", httpClient}

	project := CircleCIProject{"git", "test", "test", "token", client}

	// Sends POST request to
	// https://circleci.com/api/v1.1/project/:vcs/:owner/:project/follow?circle-token=:token
	// and returns nil on no error
	err := project.Follow()
	if err == nil {
		t.Errorf("Expected error, no error was found")
	}
}

func TestUnfollow(t *testing.T) {
	// Sends post request to
	// https://circleci.com/api/v1.1/project/:vcs/:owner/:project/unfollow?circle-token=:token
	// and reutnrs nil on no error

	// Returns error if request returns an error

	// Returns error if status code is no ok
}
