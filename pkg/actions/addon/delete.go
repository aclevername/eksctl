package addon

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/sirupsen/logrus"
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
)

func (a *Manager) Delete(addon *api.Addon) error {
	logrus.Debugf("addon: %v", addon)
	logrus.Infof("deleting addon: %s", addon.Name)
	_, err := a.clusterProvider.Provider.EKS().DeleteAddon(&eks.DeleteAddonInput{
		AddonName:   &addon.Name,
		ClusterName: &a.clusterConfig.Metadata.Name,
	})

	if err != nil {
		return fmt.Errorf("failed to delete addon %q: %v", addon.Name, err)
	}
	logrus.Infof("deleted addon: %s", addon.Name)

	stacks, err := a.stackManager.ListStacksMatching(a.makeAddonName(addon.Name))
	if err != nil {
		return fmt.Errorf("failed to list stacks: %v", err)
	}

	if len(stacks) != 0 {
		logrus.Infof("deleting associated IAM stacks")
		_, err = a.stackManager.DeleteStackByName(a.makeAddonName(addon.Name))
		if err != nil {
			return fmt.Errorf("failed to delete cloudformation stack %q: %v", a.makeAddonName(addon.Name), err)
		}
	} else {
		logrus.Infof("no associated IAM stacks found")
	}
	return nil
}
