package restore_test

import (
	"reflect"
	"testing"

	"github.com/bocklucas/dvb-made-easy/internal/restore"
)

func TestTopoSortForwardOrder(t *testing.T) {
	deps := map[string][]string{
		"app":    {"postgres", "redis"},
		"worker": {"postgres", "redis"},
	}
	services := []string{"app", "worker", "postgres", "redis"}

	order := restore.TopoSort(services, deps)

	pgIdx := indexOf(order, "postgres")
	redisIdx := indexOf(order, "redis")
	appIdx := indexOf(order, "app")
	workerIdx := indexOf(order, "worker")

	if pgIdx > appIdx {
		t.Fatalf("postgres (%d) must come before app (%d)", pgIdx, appIdx)
	}
	if redisIdx > appIdx {
		t.Fatalf("redis (%d) must come before app (%d)", redisIdx, appIdx)
	}
	if pgIdx > workerIdx {
		t.Fatalf("postgres (%d) must come before worker (%d)", pgIdx, workerIdx)
	}
	if redisIdx > workerIdx {
		t.Fatalf("redis (%d) must come before worker (%d)", redisIdx, workerIdx)
	}
}

func TestTopoSortReverseOrder(t *testing.T) {
	deps := map[string][]string{
		"app": {"postgres"},
	}
	services := []string{"app", "postgres"}

	forward := restore.TopoSort(services, deps)
	reverse := restore.ReverseOrder(forward)

	if reverse[0] != "app" {
		t.Fatalf("reverse[0]: got %q, want %q", reverse[0], "app")
	}
	if reverse[1] != "postgres" {
		t.Fatalf("reverse[1]: got %q, want %q", reverse[1], "postgres")
	}
}

func TestTopoSortNoDepsAlphabetical(t *testing.T) {
	deps := map[string][]string{}
	services := []string{"zebra", "alpha", "middle"}

	order := restore.TopoSort(services, deps)

	want := []string{"alpha", "middle", "zebra"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order: got %v, want %v", order, want)
	}
}

func TestTopoSortChainedDeps(t *testing.T) {
	deps := map[string][]string{
		"app": {"api"},
		"api": {"db"},
	}
	services := []string{"app", "api", "db"}

	order := restore.TopoSort(services, deps)

	dbIdx := indexOf(order, "db")
	apiIdx := indexOf(order, "api")
	appIdx := indexOf(order, "app")

	if dbIdx > apiIdx {
		t.Fatalf("db (%d) must come before api (%d)", dbIdx, apiIdx)
	}
	if apiIdx > appIdx {
		t.Fatalf("api (%d) must come before app (%d)", apiIdx, appIdx)
	}
}

func indexOf(slice []string, val string) int {
	for i, s := range slice {
		if s == val {
			return i
		}
	}
	return -1
}
