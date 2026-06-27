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

	cmdEnv := commandEnv(dir, step, command)
	cmd := exec.CommandContext(ctx, resolveCommandPathForEnv(command, cmdEnv), args...)
	cmd.Dir = dir
	cmd.Env = cmdEnv
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w (resolved command: %s, PATH=%q): %s", step, err, cmd.Path, envValue(cmd.Env, "PATH"), string(output))
	}

	return nil
}
