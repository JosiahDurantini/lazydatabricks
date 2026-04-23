package databricks

import (
	"context"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type TaskStatus struct {
	Key          string
	State        string
	Result       string
	StateMessage string
	ClusterID    string
	Duration     time.Duration
}

type RunDetail struct {
	RunID  int64
	JobID  int64
	State  string
	Result string
	Done   bool
	Tasks  []TaskStatus
}

func (c *Client) GetRunDetail(ctx context.Context, runID int64) (RunDetail, error) {
	run, err := c.w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return RunDetail{}, err
	}

	d := RunDetail{
		RunID: runID,
		JobID: run.JobId,
		State: string(run.State.LifeCycleState),
		Done:  isTerminalState(string(run.State.LifeCycleState)),
	}
	if run.State.ResultState != "" {
		d.Result = string(run.State.ResultState)
	}

	for _, t := range run.Tasks {
		ts := TaskStatus{
			Key:          t.TaskKey,
			State:        string(t.State.LifeCycleState),
			StateMessage: t.State.StateMessage,
		}
		if t.ClusterInstance != nil {
			ts.ClusterID = t.ClusterInstance.ClusterId
		}
		if t.State.ResultState != "" {
			ts.Result = string(t.State.ResultState)
		}
		if t.StartTime > 0 {
			start := time.UnixMilli(t.StartTime)
			end := time.Now()
			if t.EndTime > 0 {
				end = time.UnixMilli(t.EndTime)
			}
			ts.Duration = end.Sub(start).Round(time.Second)
		}
		d.Tasks = append(d.Tasks, ts)
	}
	return d, nil
}

func isTerminalState(s string) bool {
	return s == "TERMINATED" || s == "SKIPPED" || s == "INTERNAL_ERROR"
}
