package databricks

import (
	"context"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobRun struct {
	RunID     int64
	RunName   string
	JobID     int64
	Status    string
	StartTime time.Time
	Duration  time.Duration
}

// maxJobRuns caps how many recent runs the panel shows. ListRunsAll would
// page through the workspace's entire run history, so iterate and stop early.
const maxJobRuns = 25

// CancelRun cancels an in-flight job run.
func (c *Client) CancelRun(ctx context.Context, runID int64) error {
	return c.w.Jobs.CancelRunByRunId(ctx, runID)
}

func (c *Client) ListJobRuns(ctx context.Context) ([]JobRun, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	it := c.w.Jobs.ListRuns(ctx, jobs.ListRunsRequest{Limit: maxJobRuns})
	result := make([]JobRun, 0, maxJobRuns)
	for it.HasNext(ctx) && len(result) < maxJobRuns {
		r, err := it.Next(ctx)
		if err != nil {
			return nil, err
		}
		status := "UNKNOWN"
		if r.State != nil {
			status = string(r.State.LifeCycleState)
			if r.State.ResultState != "" {
				status = string(r.State.ResultState)
			}
		}

		start := time.UnixMilli(r.StartTime)
		var duration time.Duration
		if r.EndTime > 0 {
			duration = time.UnixMilli(r.EndTime).Sub(start)
		} else if r.StartTime > 0 {
			duration = time.Since(start)
		}

		name := r.RunName
		if name == "" {
			name = "Untitled run"
		}

		result = append(result, JobRun{
			RunID:     r.RunId,
			RunName:   name,
			JobID:     r.JobId,
			Status:    status,
			StartTime: start,
			Duration:  duration,
		})
	}
	return result, nil
}
