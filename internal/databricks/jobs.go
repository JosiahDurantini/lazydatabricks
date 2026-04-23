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

func (c *Client) ListJobRuns(ctx context.Context) ([]JobRun, error) {
	runs, err := c.w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{Limit: 25})
	if err != nil {
		return nil, err
	}

	result := make([]JobRun, 0, len(runs))
	for _, r := range runs {
		status := string(r.State.LifeCycleState)
		if r.State.ResultState != "" {
			status = string(r.State.ResultState)
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
