package app

import (
	"fmt"
	"testing"

	"github.com/JosiahDurantini/lazydatabricks/internal/databricks"
)

func TestFilterRunsByKey(t *testing.T) {
	runs := []databricks.JobRun{
		{RunID: 1, RunName: "Nightly ETL run"},
		{RunID: 2, RunName: "adhoc test"},
		{RunID: 3, RunName: "nightly_etl backfill"},
	}

	t.Run("case-insensitive substring", func(t *testing.T) {
		got := filterRunsByKey(runs, "ETL")
		if len(got) != 2 {
			t.Fatalf("got %d runs, want 2", len(got))
		}
		if got[0].RunID != 1 || got[1].RunID != 3 {
			t.Errorf("got runs %v, want IDs 1 and 3", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		if got := filterRunsByKey(runs, "missing"); len(got) != 0 {
			t.Errorf("got %d runs, want 0", len(got))
		}
	})

	t.Run("capped at five", func(t *testing.T) {
		var many []databricks.JobRun
		for i := 0; i < 10; i++ {
			many = append(many, databricks.JobRun{RunID: int64(i), RunName: fmt.Sprintf("etl %d", i)})
		}
		if got := filterRunsByKey(many, "etl"); len(got) != 5 {
			t.Errorf("got %d runs, want cap of 5", len(got))
		}
	})
}
