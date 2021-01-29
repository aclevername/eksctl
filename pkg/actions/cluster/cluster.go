package cluster

import (
	"time"

	"github.com/sirupsen/logrus"
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/eks"
)

type Cluster interface {
	Upgrade(dryRun bool) error
	Delete(waitInterval time.Duration, wait bool) error
}

func New(cfg *api.ClusterConfig, ctl *eks.ClusterProvider) (Cluster, error) {
	if err := ctl.RefreshClusterStatus(cfg); err != nil {
		return nil, err
	}

	stackManager := ctl.NewStackManager(cfg)
	hasClusterStack, err := stackManager.HasClusterStack()
	if err != nil {
		return nil, err
	}

	if hasClusterStack {
		logrus.Debugf("Cluster %q was created by eksctl", cfg.Metadata.Name)
		return NewOwnedCluster(cfg, ctl, stackManager)
	}
	logrus.Debugf("Cluster %q was not created by eksctl", cfg.Metadata.Name)

	clientSet, err := ctl.NewStdClientSet(cfg)
	if err != nil {
		return nil, err
	}
	return NewUnownedCluster(cfg, ctl, clientSet, stackManager), nil
}
