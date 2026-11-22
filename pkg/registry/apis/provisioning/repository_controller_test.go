package provisioning

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProvideRepositoryController(t *testing.T) {
	t.Run("provisioning disabled", func(t *testing.T) {
		require.NotNil(t, ProvideRepositoryController(nil))
	})

	t.Run("starts once the API server is ready", func(t *testing.T) {
		b := &APIBuilder{}
		require.NotNil(t, ProvideRepositoryController(b))
		require.Len(t, b.serverReadyHooks, 1)
	})
}
