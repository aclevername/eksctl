package nodegroup

import (
	"fmt"

	awseks "github.com/aws/aws-sdk-go/service/eks"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/eks/eksiface"

	"github.com/weaveworks/eksctl/pkg/eks"

	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
)

func (m *Manager) Delete(nodeGroups []*api.NodeGroup, managedNodeGroups []*api.ManagedNodeGroup, wait, plan bool) error {
	var nodeGroupsWithStacks []eks.KubeNodeGroup
	var nodeGroupsWithoutStacksDeleteTasks []*DeleteUnownedNodegroupTask

	for _, n := range nodeGroups {
		nodeGroupsWithStacks = append(nodeGroupsWithStacks, n)
	}

	for _, n := range managedNodeGroups {
		hasStacks, err := m.hasStacks(n.Name)
		if err != nil {
			return err
		}

		if hasStacks {
			nodeGroupsWithStacks = append(nodeGroupsWithStacks, n)
		} else {
			nodeGroupsWithoutStacksDeleteTasks = append(nodeGroupsWithoutStacksDeleteTasks, &DeleteUnownedNodegroupTask{
				cluster:   m.cfg.Metadata.Name,
				nodegroup: n.Name,
				eksAPI:    m.ctl.Provider.EKS(),
				info:      fmt.Sprintf("delete unowned nodegroup %q", n.Name),
			})
		}
	}

	shouldDelete := func(ngName string) bool {
		for _, n := range nodeGroupsWithStacks {
			if n.NameString() == ngName {
				return true
			}
		}
		return false
	}

	tasks, err := m.manager.NewTasksToDeleteNodeGroups(shouldDelete, wait, nil)
	if err != nil {
		return err
	}

	for _, t := range nodeGroupsWithoutStacksDeleteTasks {
		tasks.Append(t)
	}

	tasks.PlanMode = plan
	logrus.Infof(tasks.Describe())
	if errs := tasks.DoAllSync(); len(errs) > 0 {
		return handleErrors(errs, "nodegroup(s)")
	}
	return nil
}

type DeleteUnownedNodegroupTask struct {
	info      string
	eksAPI    eksiface.EKSAPI
	cluster   string
	nodegroup string
}

func (d *DeleteUnownedNodegroupTask) Describe() string {
	return d.info
}

func (d *DeleteUnownedNodegroupTask) Do(errorchan chan error) error {
	out, err := d.eksAPI.DeleteNodegroup(&awseks.DeleteNodegroupInput{
		ClusterName:   &d.cluster,
		NodegroupName: &d.nodegroup,
	})
	go func() {
		errorchan <- err
	}()

	if out != nil {
		logrus.Debugf("Delete nodegroup %q output: %s", d.nodegroup, out.String())
	}
	return err
}

func handleErrors(errs []error, subject string) error {
	logrus.Infof("%d error(s) occurred while deleting %s", len(errs), subject)
	for _, err := range errs {
		logrus.Errorf("%s\n", err.Error())
	}
	return fmt.Errorf("failed to delete %s", subject)
}
