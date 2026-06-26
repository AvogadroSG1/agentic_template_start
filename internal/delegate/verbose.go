package delegate

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

type VerboseRunner struct {
	out   io.Writer
	isTTY bool
}

func NewVerboseRunner(w io.Writer) *VerboseRunner {
	tty := false
	if f, ok := w.(*os.File); ok {
		tty = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return &VerboseRunner{out: w, isTTY: tty}
}

func (v *VerboseRunner) Run(ctx context.Context, dir string, step string, command string, args ...string) error {
	fmt.Fprintf(v.out, "→ %s\n", step)
	start := time.Now()

	if err := prepareCommandEnv(dir, step, command); err != nil {
		fmt.Fprintf(v.out, "→ %s ✗ (%.1fs)\n", step, time.Since(start).Seconds())
		return fmt.Errorf("%s: %w", step, err)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.Env = commandEnv(dir, step, command)

	lw := &lineWriter{out: v.out}
	cmd.Stdout = lw
	cmd.Stderr = lw

	err := cmd.Run()
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(v.out, "→ %s ✗ (%.1fs)\n", step, elapsed.Seconds())
		return fmt.Errorf("%s: %w", step, err)
	}

	if v.isTTY && lw.lines > 0 {
		// Move cursor up (output lines + header) and clear to end of screen
		fmt.Fprintf(v.out, "\033[%dA\033[J", lw.lines+1)
	} else if v.isTTY {
		// No output lines, just erase the header
		fmt.Fprintf(v.out, "\033[1A\033[J")
	}

	fmt.Fprintf(v.out, "→ %s ✓ (%.1fs)\n", step, elapsed.Seconds())
	return nil
}

type lineWriter struct {
	out     io.Writer
	mu      sync.Mutex
	lines   int
	atStart bool
}

func (lw *lineWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	n := len(p)
	for len(p) > 0 {
		if lw.atStart || lw.lines == 0 {
			lw.out.Write([]byte("  "))
			if lw.lines == 0 {
				lw.lines = 1
			}
			lw.atStart = false
		}

		// Find next newline
		idx := -1
		for i, b := range p {
			if b == '\n' {
				idx = i
				break
			}
		}

		if idx >= 0 {
			lw.out.Write(p[:idx+1])
			p = p[idx+1:]
			lw.lines++
			lw.atStart = true
		} else {
			lw.out.Write(p)
			p = nil
		}
	}

	return n, nil
}
