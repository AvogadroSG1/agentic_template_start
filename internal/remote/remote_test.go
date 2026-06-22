package remote

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mkproj/internal/project"
)

func TestPublishRemoteNoneOnlyCommitsLocally(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	err := Publish(context.Background(), runner, t.TempDir(), PublishOptions{Remote: project.RemoteNone})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	want := []string{"git add", "git commit"}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("steps = %#v, want %#v", got, want)
	}
}

func TestPublishRemoteURLRequiresAURL(t *testing.T) {
	t.Parallel()

	err := Publish(context.Background(), &recordingRunner{}, t.TempDir(), PublishOptions{Remote: project.RemoteURL})
	if err == nil || !strings.Contains(err.Error(), "remote url is required") {
		t.Fatalf("Publish() error = %v, want missing URL error", err)
	}
}

func TestPublishRemoteGHReportsPushRetryAdvice(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{failStep: "git push"}
	err := Publish(context.Background(), runner, t.TempDir(), PublishOptions{Remote: project.RemoteGH, RepoName: "sample-app"})
	if err == nil {
		t.Fatal("Publish() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "remote created but initial push failed") {
		t.Fatalf("Publish() error = %v, want push retry guidance", err)
	}
}

type recordingRunner struct {
	failStep string
	steps    []string
}

func (r *recordingRunner) Run(_ context.Context, _ string, step string, _ string, _ ...string) error {
	r.steps = append(r.steps, step)
	if step == r.failStep {
		return errors.New("boom")
	}

	return nil
}

func (r *recordingRunner) stepNames() []string {
	return append([]string(nil), r.steps...)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
