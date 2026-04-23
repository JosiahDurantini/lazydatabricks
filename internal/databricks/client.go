package databricks

import (
	"github.com/databricks/databricks-sdk-go"
)

// Client wraps the Databricks workspace client.
// Auth is handled automatically via environment variables or ~/.databrickscfg.
type Client struct {
	w *databricks.WorkspaceClient
}

func NewClient() (*Client, error) {
	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		return nil, err
	}
	return &Client{w: w}, nil
}
