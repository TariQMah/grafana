package provisioning

import (
	"time"

	"github.com/grafana/grafana-app-sdk/logging"
	"k8s.io/client-go/tools/cache"

	provisioningapis "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/controller"
	keysapi "github.com/grafana/grafana/pkg/services/apiserver/keys"
)

const defaultWebhookSecretRotationInterval = 30 * 24 * time.Hour

// RepositoryControllerRegisterer is set to satisfy wire gen and make sure
// ProvideRepositoryController is called.
type RepositoryControllerRegisterer struct{}

// ProvideRepositoryController runs the repository controller once the
// provisioning API server is ready. It depends on the APIBuilder returned by
// RegisterAPIService, whose post-start hook is what signals readiness; the
// builder is nil when provisioning is disabled.
func ProvideRepositoryController(builder *APIBuilder) *RepositoryControllerRegisterer {
	if builder != nil {
		builder.onServerReady(builder.startRepositoryController)
	}
	return &RepositoryControllerRegisterer{}
}

func (b *APIBuilder) startRepositoryController(ready serverReady) error {
	webhookSecretRotationInterval := b.webhookSecretRotationInterval
	if webhookSecretRotationInterval <= 0 {
		webhookSecretRotationInterval = defaultWebhookSecretRotationInterval
	}

	var repoKeys keysapi.Lister
	if b.keysOnlyReList {
		gvr := provisioningapis.RepositoryResourceInfo.GroupVersionResource()
		repoKeys = keysapi.NewGRPCLister(b.unified, gvr)
		logging.FromContext(ready.ctx).Info("provisioning re-list will ask for keys only", "transport", "storage", "resource", gvr.Resource)
	}

	repoController, hasSynced, err := controller.SetupRepositoryController(ready.ctx, controller.RepositoryControllerDeps{
		ProvisioningClient:            ready.clientset,
		NATSSubscriber:                b.natsSubscriber,
		KeysLister:                    repoKeys,
		RelistProjectionMetrics:       ready.relistProjectionMetrics,
		FreshnessMetrics:              ready.freshnessMetrics,
		RepoFactory:                   b.repoFactory,
		ConnectionFactory:             b.connectionFactory,
		ResourceLister:                b.resourceLister,
		Clients:                       b.clients,
		Jobs:                          b.jobs,
		HealthChecker:                 b.GetHealthChecker(),
		StatusPatcher:                 b.statusPatcher,
		QuotaGetter:                   b.quotaGetter,
		IncrementalPolicy:             b.incrementalPolicy,
		Registry:                      b.registry,
		Tracer:                        b.tracer,
		ParallelOperations:            10,
		ResyncInterval:                ready.resyncInterval,
		MinSyncInterval:               b.minSyncInterval,
		DrainTimeout:                  30 * time.Second,
		WebhookSecretRotationInterval: webhookSecretRotationInterval,
	})
	if err != nil {
		return err
	}

	// Wait for the cache to sync off the hot path so we don't block apiserver
	// startup; the controller only starts its workers once its handler has
	// processed the initial list.
	go func() {
		if !cache.WaitForCacheSync(ready.ctx.Done(), hasSynced) {
			return
		}
		repoController.Run(ready.ctx, repoControllerWorkers, func() {}, func() {})
	}()
	return nil
}
