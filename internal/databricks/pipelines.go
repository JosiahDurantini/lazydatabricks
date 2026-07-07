package databricks

import (
	"context"
	"time"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type PipelineInfo struct {
	PipelineID       string
	Name             string
	State            string
	Health           string
	CreatorUserName  string
	ClusterID        string
	LatestUpdateID   string
	LatestUpdate     string
	LatestUpdateTime time.Time
}

// StopPipeline stops the pipeline's active update; the pipeline returns to IDLE.
func (c *Client) StopPipeline(ctx context.Context, pipelineID string) error {
	// The waiter is discarded — the panel refreshes rather than blocking.
	_, err := c.w.Pipelines.Stop(ctx, pipelines.StopRequest{PipelineId: pipelineID})
	return err
}

func (c *Client) ListPipelines(ctx context.Context) ([]PipelineInfo, error) {
	infos, err := c.w.Pipelines.ListPipelinesAll(ctx, pipelines.ListPipelinesRequest{MaxResults: 100})
	if err != nil {
		return nil, err
	}
	out := make([]PipelineInfo, 0, len(infos))
	for _, p := range infos {
		info := PipelineInfo{
			PipelineID:      p.PipelineId,
			Name:            p.Name,
			State:           string(p.State),
			Health:          string(p.Health),
			CreatorUserName: p.CreatorUserName,
			ClusterID:       p.ClusterId,
		}
		// LatestUpdates is ordered newest first.
		if len(p.LatestUpdates) > 0 {
			u := p.LatestUpdates[0]
			info.LatestUpdateID = u.UpdateId
			info.LatestUpdate = string(u.State)
			if t, err := time.Parse(time.RFC3339, u.CreationTime); err == nil {
				info.LatestUpdateTime = t
			}
		}
		out = append(out, info)
	}
	return out, nil
}
