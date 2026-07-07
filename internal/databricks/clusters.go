package databricks

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

type ClusterInfo struct {
	ClusterID              string
	ClusterName            string
	State                  string
	StateMessage           string
	NodeTypeID             string
	NumWorkers             int
	SparkVersion           string
	CreatorUserName        string
	AutoterminationMinutes int
}

func clusterInfoFromDetails(cl compute.ClusterDetails) ClusterInfo {
	return ClusterInfo{
		ClusterID:              cl.ClusterId,
		ClusterName:            cl.ClusterName,
		State:                  string(cl.State),
		StateMessage:           cl.StateMessage,
		NodeTypeID:             cl.NodeTypeId,
		NumWorkers:             cl.NumWorkers,
		SparkVersion:           cl.SparkVersion,
		CreatorUserName:        cl.CreatorUserName,
		AutoterminationMinutes: cl.AutoterminationMinutes,
	}
}

func (c *Client) GetCluster(ctx context.Context, clusterID string) (ClusterInfo, error) {
	cl, err := c.w.Clusters.Get(ctx, compute.GetClusterRequest{ClusterId: clusterID})
	if err != nil {
		return ClusterInfo{}, err
	}
	return clusterInfoFromDetails(*cl), nil
}

func (c *Client) ListClusters(ctx context.Context) ([]ClusterInfo, error) {
	details, err := c.w.Clusters.ListAll(ctx, compute.ListClustersRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]ClusterInfo, 0, len(details))
	for _, cl := range details {
		out = append(out, clusterInfoFromDetails(cl))
	}
	return out, nil
}

func (c *Client) StartCluster(ctx context.Context, clusterID string) error {
	// The waiter is discarded — the panel refreshes rather than blocking.
	_, err := c.w.Clusters.Start(ctx, compute.StartCluster{ClusterId: clusterID})
	return err
}

// StopCluster terminates (does not permanently delete) the cluster.
func (c *Client) StopCluster(ctx context.Context, clusterID string) error {
	_, err := c.w.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: clusterID})
	return err
}
