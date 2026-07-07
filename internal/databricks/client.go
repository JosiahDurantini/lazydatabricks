package databricks

import (
	"net/http"

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
	// The SDK resolves credentials lazily; force it now so a misconfigured
	// environment fails fast with a clear error instead of inside every panel.
	if err := w.Config.Authenticate(&http.Request{Header: make(http.Header)}); err != nil {
		return nil, err
	}
	return &Client{w: w}, nil
}
