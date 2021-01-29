package defaultaddons

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/weaveworks/eksctl/pkg/addons"
	"github.com/weaveworks/eksctl/pkg/printers"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// KubeProxy is the name of the kube-proxy addon
	KubeProxy = "kube-proxy"
)

func IsKubeProxyUpToDate(clientSet kubernetes.Interface, controlPlaneVersion string) (bool, error) {
	d, err := clientSet.AppsV1().DaemonSets(metav1.NamespaceSystem).Get(context.TODO(), KubeProxy, metav1.GetOptions{})
	if err != nil {
		if apierrs.IsNotFound(err) {
			logrus.Warningf("%q was not found", KubeProxy)
			return true, nil
		}
		return false, errors.Wrapf(err, "getting %q", KubeProxy)
	}
	if numContainers := len(d.Spec.Template.Spec.Containers); !(numContainers >= 1) {
		return false, fmt.Errorf("%s has %d containers, expected at least 1", KubeProxy, numContainers)
	}

	desiredTag := kubeProxyImageTag(controlPlaneVersion)
	image := d.Spec.Template.Spec.Containers[0].Image
	imageTag, err := addons.ImageTag(image)
	if err != nil {
		return false, err
	}
	return desiredTag == imageTag, nil
}

// UpdateKubeProxyImageTag updates image tag for kube-system:daemonset/kube-proxy based to match controlPlaneVersion
func UpdateKubeProxyImageTag(clientSet kubernetes.Interface, controlPlaneVersion string, plan bool) (bool, error) {
	printer := printers.NewJSONPrinter()

	d, err := clientSet.AppsV1().DaemonSets(metav1.NamespaceSystem).Get(context.TODO(), KubeProxy, metav1.GetOptions{})
	if err != nil {
		if apierrs.IsNotFound(err) {
			logrus.Warningf("%q was not found", KubeProxy)
			return false, nil
		}
		return false, errors.Wrapf(err, "getting %q", KubeProxy)
	}
	if numContainers := len(d.Spec.Template.Spec.Containers); !(numContainers >= 1) {
		return false, fmt.Errorf("%s has %d containers, expected at least 1", KubeProxy, numContainers)
	}

	if err := printer.LogObj(logrus.Debugf, KubeProxy+" [current] = \\\n%s\n", d); err != nil {
		return false, err
	}

	image := &d.Spec.Template.Spec.Containers[0].Image
	imageParts := strings.Split(*image, ":")

	if len(imageParts) != 2 {
		return false, fmt.Errorf("unexpected image format %q for %q", *image, KubeProxy)
	}

	desiredTag := kubeProxyImageTag(controlPlaneVersion)

	if imageParts[1] == desiredTag {
		logrus.Debugf("imageParts = %v, desiredTag = %s", imageParts, desiredTag)
		logrus.Infof("%q is already up-to-date", KubeProxy)
		return false, nil
	}

	if plan {
		logrus.Errorf("(plan) %q is not up-to-date", KubeProxy)
		return true, nil
	}

	imageParts[1] = desiredTag
	*image = strings.Join(imageParts, ":")

	if err := printer.LogObj(logrus.Debugf, KubeProxy+" [updated] = \\\n%s\n", d); err != nil {
		return false, err
	}
	if _, err := clientSet.AppsV1().DaemonSets(metav1.NamespaceSystem).Update(context.TODO(), d, metav1.UpdateOptions{}); err != nil {
		return false, err
	}

	logrus.Infof("%q is now up-to-date", KubeProxy)
	return false, nil
}

func kubeProxyImageTag(controlPlaneVersion string) string {
	return fmt.Sprintf("v%s-eksbuild.1", controlPlaneVersion)
}
