package allowlist

import (
	"fmt"
	"os"
	"strings"
)

const Version = 1

const (
	beginMarker = "// BEGIN MKPROJ ALLOW v:"
	endMarker   = "// END MKPROJ ALLOW"
)

type Status struct {
	CurrentVersion int
	Embedded       int
	Stale          bool
}

func Sync(path string, block string, checkOnly bool) (Status, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Status{}, err
	}

	status := Detect(string(data))
	if checkOnly {
		return status, nil
	}

	start := strings.Index(string(data), beginMarker)
	end := strings.Index(string(data), endMarker)
	if start == -1 || end == -1 || end < start {
		return Status{}, fmt.Errorf("managed block markers not found in %s", path)
	}

	replacement := fmt.Sprintf("%s%d\n%s\n%s", beginMarker, Version, block, endMarker)
	updated := string(data[:start]) + replacement + string(data[end+len(endMarker):])

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return Status{}, err
	}

	return Detect(updated), nil
}

func Detect(contents string) Status {
	start := strings.Index(contents, beginMarker)
	if start == -1 {
		return Status{Embedded: Version}
	}

	start += len(beginMarker)
	end := strings.Index(contents[start:], "\n")
	if end == -1 {
		return Status{Embedded: Version}
	}

	var current int
	_, _ = fmt.Sscanf(contents[start:start+end], "%d", &current)

	return Status{
		CurrentVersion: current,
		Embedded:       Version,
		Stale:          current < Version,
	}
}
