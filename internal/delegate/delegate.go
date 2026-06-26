package delegate

import (
	"context"
	"fmt"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, dir string, step string, command string, args ...string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir string, step string, command string, args ...string) error {
	if err := prepareCommandEnv(dir, step, command); err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.Env = commandEnv(dir, step, command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", step, err, string(output))
	}

	return nil
}
