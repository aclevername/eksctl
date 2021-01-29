package addon

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/weaveworks/eksctl/pkg/cfn/manager"

	"github.com/weaveworks/eksctl/pkg/cfn/builder"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
)

func (a *Manager) Update(addon *api.Addon) error {
	logrus.Debugf("addon: %v", addon)

	updateAddonInput := &eks.UpdateAddonInput{
		AddonName:   &addon.Name,
		ClusterName: &a.clusterConfig.Metadata.Name,
		//ResolveConflicts: 		"enum":["OVERWRITE","NONE"]
	}

	if addon.Force {
		updateAddonInput.ResolveConflicts = aws.String("overwrite")
		logrus.Debugf("setting resolve conflicts to overwrite")

	}

	summary, err := a.Get(addon)
	if err != nil {
		return err
	}

	if addon.Version == "" {
		// preserve existing version
		// Might be redundant, does the API care?
		logrus.Infof("no new version provided, preserving existing version: %s", summary.Version)

		updateAddonInput.AddonVersion = &summary.Version
	} else {
		if summary.Version != addon.Version {
			logrus.Infof("new version provided %s", addon.Version)
		}
		updateAddonInput.AddonVersion = &addon.Version
	}

	//check if we have been provided a different set of policies/role
	if addon.ServiceAccountRoleARN != "" {
		updateAddonInput.ServiceAccountRoleArn = &addon.ServiceAccountRoleARN
	} else if addon.AttachPolicy != nil || (addon.AttachPolicyARNs != nil && len(addon.AttachPolicyARNs) != 0) {
		serviceAccountRoleARN, err := a.updateWithNewPolicies(addon)
		if err != nil {
			return err
		}
		updateAddonInput.ServiceAccountRoleArn = &serviceAccountRoleARN
	} else {
		//preserve current role
		if summary.IAMRole != "" {
			updateAddonInput.ServiceAccountRoleArn = &summary.IAMRole
		}
	}

	logrus.Infof("updating addon")
	logrus.Debugf(updateAddonInput.String())

	output, err := a.clusterProvider.Provider.EKS().UpdateAddon(updateAddonInput)
	if err != nil {
		return fmt.Errorf("failed to update addon %q: %v", addon.Name, err)
	}
	if output != nil {
		logrus.Debugf(output.String())
	}
	return nil
}

func (a *Manager) updateWithNewPolicies(addon *api.Addon) (string, error) {
	stackName := a.makeAddonName(addon.Name)
	existingStacks, err := a.stackManager.ListStacksMatching(stackName)
	if err != nil {
		return "", err
	}

	namespace, serviceAccount := a.getKnownServiceAccountLocation(addon)

	if len(existingStacks) == 0 {
		return a.createNewRole(addon, namespace, serviceAccount)
	}

	createNewTemplate, err := a.createNewTemplate(addon, namespace, serviceAccount)
	if err != nil {
		return "", err
	}
	var templateBody manager.TemplateBody = createNewTemplate
	err = a.stackManager.UpdateStack(stackName, "updating-policy", "updating policies", templateBody, nil)
	if err != nil {
		return "", err
	}

	existingStacks, err = a.stackManager.ListStacksMatching(stackName)
	if err != nil {
		return "", err
	}

	return *existingStacks[0].Outputs[0].OutputValue, nil
}

func (a *Manager) createNewTemplate(addon *api.Addon, namespace, serviceAccount string) ([]byte, error) {

	var resourceSet *builder.IAMRoleResourceSet
	if addon.AttachPolicyARNs != nil && len(addon.AttachPolicyARNs) != 0 {
		resourceSet = builder.NewIAMRoleResourceSetWithAttachPolicyARNs(addon.Name, namespace, serviceAccount, addon.AttachPolicyARNs, a.oidcManager)
		err := resourceSet.AddAllResources()
		if err != nil {
			return []byte(""), err
		}
	} else {
		resourceSet = builder.NewIAMRoleResourceSetWithAttachPolicy(addon.Name, namespace, serviceAccount, addon.AttachPolicy, a.oidcManager)
		err := resourceSet.AddAllResources()
		if err != nil {
			return []byte(""), err
		}
	}

	return resourceSet.RenderJSON()
}

func (a *Manager) createNewRole(addon *api.Addon, namespace, serviceAccount string) (string, error) {
	if addon.AttachPolicyARNs != nil && len(addon.AttachPolicyARNs) != 0 {
		logrus.Infof("creating role using provided policies ARNs")
		return a.createRoleUsingAttachPolicyARNs(addon, namespace, serviceAccount)
	}

	logrus.Infof("creating role using provided policies")
	return a.createRoleUsingAttachPolicy(addon, namespace, serviceAccount)
}
