package task

import (
	"context"
	"strings"

	"github.com/dimaskiddo/play-with-docker/event"
	"github.com/dimaskiddo/play-with-docker/k8s"
	"github.com/dimaskiddo/play-with-docker/pwd/types"
)

type checkK8sClusterStatusTask struct {
	event   event.EventApi
	factory k8s.FactoryApi
}

var CheckK8sStatusEvent event.EventType

func init() {
	CheckK8sStatusEvent = event.EventType("instance k8s status")
}

func NewCheckK8sClusterStatus(e event.EventApi, f k8s.FactoryApi) *checkK8sClusterStatusTask {
	return &checkK8sClusterStatusTask{event: e, factory: f}
}

func (c *checkK8sClusterStatusTask) Name() string {
	return "CheckK8sClusterStatus"
}

func (c checkK8sClusterStatusTask) Run(ctx context.Context, i *types.Instance) error {
	// Skip if this is not a Kubernetes instance (e.g., regular Docker Swarm)
	// We identify this by checking if the image contains "k8s" or if the kubelet is available
	if !strings.Contains(strings.ToLower(i.Image), "k8s") && !strings.Contains(strings.ToLower(i.Image), "kubernetes") {
		// This is likely a Docker Swarm instance, skip kubernetes checks
		return nil
	}

	status := ClusterStatus{Instance: i.Name}

	kc, err := c.factory.GetKubeletForInstance(i)
	if err != nil {
		// If kubelet is not available, this is not a k8s instance, skip silently
		return nil
	}

	if isManager, err := kc.IsManager(); err != nil {
		c.event.Emit(CheckSwarmStatusEvent, i.SessionId, status)
		return err
	} else if !isManager {
		status.IsWorker = true
	} else {
		status.IsManager = true
	}

	c.event.Emit(CheckK8sStatusEvent, i.SessionId, status)

	return nil
}
