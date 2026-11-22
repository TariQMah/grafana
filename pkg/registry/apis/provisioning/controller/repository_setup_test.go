package controller

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	provisioning "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/apps/provisioning/pkg/generated/clientset/versioned/fake"
)

func TestSetupRepositoryController_EnqueuesExistingRepositoriesOnSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	client := fake.NewClientset(&provisioning.Repository{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "repo-a"},
	})

	rc, hasSynced, err := SetupRepositoryController(ctx, RepositoryControllerDeps{
		ProvisioningClient: client,
		Registry:           prometheus.NewRegistry(),
		ParallelOperations: 1,
		ResyncInterval:     time.Minute,
	})
	require.NoError(t, err)

	require.True(t, cache.WaitForCacheSync(ctx.Done(), hasSynced), "source should sync once the initial list is delivered")
	require.Equal(t, 1, rc.queue.Len())
	key, _ := rc.queue.Get()
	require.Equal(t, "default/repo-a", key)
}
