package fs

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"golang.org/x/net/context"

	"strconv"
	"strings"

	"sourcegraph.com/sourcegraph/sourcegraph/conf"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/server/accesscontrol"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
)

// BuildLogs is a local FS-backed implementation of the BuildLogs
// store.
//
// TODO(sqs): use the same dir as the other services? right now this
// uses conf.BuildLogDir, which is weird and inconsistent.
type BuildLogs struct{}

var _ store.BuildLogs = (*BuildLogs)(nil)

func (s *BuildLogs) Get(ctx context.Context, task sourcegraph.TaskSpec, minIDStr string, minTime, maxTime time.Time) (*sourcegraph.LogEntries, error) {
	if err := accesscontrol.VerifyUserHasReadAccess(ctx, "BuildLogs.Get", task.Build.Repo.URI); err != nil {
		return nil, err
	}
	// Read the log file.
	b, err := ioutil.ReadFile(logFilePath(task))
	if err != nil {
		if os.IsNotExist(err) {
			return &sourcegraph.LogEntries{}, nil
		}
		return nil, err
	}
	var lines []string
	if len(b) == 0 {
		return &sourcegraph.LogEntries{}, nil
	}
	if len(b) > 0 {
		lines = strings.Split(string(b), "\n")
	}
	minID, err := strconv.Atoi(minIDStr)
	if err != nil && minIDStr != "" {
		return nil, err
	}
	if minID < 0 {
		minID = 0
	} else if minID > len(lines) {
		minID = len(lines)
	}
	return &sourcegraph.LogEntries{Entries: lines[minID:], MaxID: strconv.Itoa(len(lines) - 1)}, nil
}

// logFilePath returns the filename to use for the log file for the
// given task.
func logFilePath(task sourcegraph.TaskSpec) string {
	p := task.IDString() + ".log"

	// Clean the path of any relative parts so that we refuse to read files
	// outside the build log dir.
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	p = path.Clean(p)[1:]

	return filepath.Join(conf.BuildLogDir, p)
}
