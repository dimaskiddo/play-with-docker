package task

import (
	"context"
	"log"
	"strings"

	"github.com/dimaskiddo/play-with-docker/event"
	"github.com/dimaskiddo/play-with-docker/k8s"
	"github.com/dimaskiddo/play-with-docker/pwd/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type checkK8sClusterExposedPortsTask struct {
	event   event.EventApi
	factory k8s.FactoryApi
}

var CheckK8sClusterExpoedPortsEvent event.EventType

func init() {
	CheckK8sClusterExpoedPortsEvent = event.EventType("instance k8s cluster ports")
}

func (t *checkK8sClusterExposedPortsTask) Name() string {
	return "CheckK8sClusterPorts"
}

func NewCheckK8sClusterExposedPorts(e event.EventApi, f k8s.FactoryApi) *checkK8sClusterExposedPortsTask {
	return &checkK8sClusterExposedPortsTask{event: e, factory: f}
}

func (c checkK8sClusterExposedPortsTask) Run(ctx context.Context, i *types.Instance) error {
	// Skip if this is not a Kubernetes instance (e.g., regular Docker Swarm)
	// We identify this by checking if the image contains "k8s" or if the kubelet is available
	if !strings.Contains(strings.ToLower(i.Image), "k8s") && !strings.Contains(strings.ToLower(i.Image), "kubernetes") {
		return nil
	}

	kc, err := c.factory.GetKubeletForInstance(i)
	if err != nil {
		// If kubelet is not available, this is not a k8s instance, skip silently
		return nil
	}

	if isManager, err := kc.IsManager(); err != nil {
		log.Println(err)
		return err
	} else if !isManager {
		return nil
	}

	k8s, err := c.factory.GetForInstance(i)
	if err != nil {
		log.Println(err)
		return err
	}

	list, err := k8s.CoreV1().Services("").List(ctx, meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	exposedPorts := []int{}

	for _, svc := range list.Items {
		for _, p := range svc.Spec.Ports {
			if p.NodePort > 0 {
				exposedPorts = append(exposedPorts, int(p.NodePort))
			}
		}
	}

	nodeList, err := k8s.CoreV1().Nodes().List(ctx, meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	instances := []string{}
	for _, node := range nodeList.Items {
		instances = append(instances, node.Name)
	}

	c.event.Emit(CheckSwarmPortsEvent, i.SessionId, ClusterPorts{Manager: i.Name, Instances: instances, Ports: exposedPorts})

	return nil
}
