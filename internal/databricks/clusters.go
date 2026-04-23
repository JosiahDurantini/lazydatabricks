package databricks

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ClusterInfo struct {
	ClusterID   string
	ClusterName string
	State       string
	NodeTypeID  string
}

func (c *Client) GetCluster(ctx context.Context, clusterID string) (ClusterInfo, error) {
	cl, err := c.w.Clusters.Get(ctx, compute.GetClusterRequest{ClusterId: clusterID})
	if err != nil {
		return ClusterInfo{}, err
	}
	return ClusterInfo{
		ClusterID:   clusterID,
		ClusterName: cl.ClusterName,
		State:       string(cl.State),
		NodeTypeID:  cl.NodeTypeId,
	}, nil
}
