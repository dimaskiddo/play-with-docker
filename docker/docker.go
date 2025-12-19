package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"github.com/containerd/containerd/reference"
	"github.com/dimaskiddo/play-with-docker/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

const (
	Byte     = 1
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
)

type DockerApi interface {
	GetClient() *client.Client

	NetworkCreate(id string, opts types.NetworkCreate) error
	NetworkConnect(container, network, ip string, aliases []string) (string, error)
	NetworkInspect(id string) (types.NetworkResource, error)
	NetworkDelete(id string) error
	NetworkDisconnect(containerId, networkId string) error

	DaemonInfo() (types.Info, error)
	DaemonHost() string

	GetSwarmPorts() ([]string, []uint16, error)
	GetPorts() ([]uint16, error)

	ContainerStats(name string) (io.ReadCloser, error)
	ContainerResize(name string, rows, cols uint) error
	ContainerRename(old, new string) error
	ContainerDelete(name string) error
	ContainerCreate(opts CreateContainerOpts) error
	ContainerIPs(id string) (map[string]string, error)
	ExecAttach(instanceName string, command []string, out io.Writer) (int, error)
	Exec(instanceName string, command []string) (int, error)

	CreateAttachConnection(name string) (net.Conn, error)
	CopyToContainer(containerName, destination, fileName string, content io.Reader) error
	CopyFromContainer(containerName, filePath string) (io.Reader, error)
	SwarmInit(advertiseAddr string) (*SwarmTokens, error)
	SwarmJoin(addr, token string) error

	ConfigCreate(name string, labels map[string]string, data []byte) error
	ConfigDelete(name string) error

	VolumeCreate(name string, labels map[string]string) error
	VolumeInspect(name string) (types.Volume, error)
}

type SwarmTokens struct {
	Manager string
	Worker  string
}

type docker struct {
	c *client.Client
}

func (d *docker) GetClient() *client.Client {
	return d.c
}

func (d *docker) ConfigCreate(name string, labels map[string]string, data []byte) error {
	config := swarm.ConfigSpec{}
	config.Name = name
	config.Labels = labels
	config.Data = data
	_, err := d.c.ConfigCreate(context.Background(), config)

	return err
}

func (d *docker) ConfigDelete(name string) error {
	return d.c.ConfigRemove(context.Background(), name)
}

func (d *docker) NetworkCreate(id string, opts types.NetworkCreate) error {
	_, err := d.c.NetworkCreate(context.Background(), id, opts)

	if err != nil {
		log.Printf("Starting session err [%s]\n", err)
		return err
	}

	return nil
}

func (d *docker) NetworkConnect(containerId, networkId, ip string, aliases []string) (string, error) {
	settings := &network.EndpointSettings{
		Aliases: aliases,
	}

	if ip != "" {
		settings.IPAddress = ip
	}

	err := d.c.NetworkConnect(context.Background(), networkId, containerId, settings)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Connection container to network err [%s]\n", err)

		return "", err
	}

	container, err := d.c.ContainerInspect(context.Background(), containerId)
	if err != nil {
		return "", err
	}

	n, found := container.NetworkSettings.Networks[networkId]
	if !found {
		return "", fmt.Errorf("Container [%s] connected to the network [%s] but couldn't obtain it's IP address", containerId, networkId)
	}

	return n.IPAddress, nil
}

func (d *docker) NetworkInspect(id string) (types.NetworkResource, error) {
	return d.c.NetworkInspect(context.Background(), id, types.NetworkInspectOptions{})
}

func (d *docker) DaemonInfo() (types.Info, error) {
	return d.c.Info(context.Background())
}

func (d *docker) DaemonHost() string {
	return d.c.DaemonHost()
}

func (d *docker) GetSwarmPorts() ([]string, []uint16, error) {
	hosts := []string{}
	ports := []uint16{}

	nodesIdx := map[string]string{}
	nodes, nodesErr := d.c.NodeList(context.Background(), types.NodeListOptions{})
	if nodesErr != nil {
		return nil, nil, nodesErr
	}

	for _, n := range nodes {
		nodesIdx[n.ID] = n.Description.Hostname
		hosts = append(hosts, n.Description.Hostname)
	}

	services, err := d.c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return nil, nil, err
	}

	for _, service := range services {
		for _, p := range service.Endpoint.Ports {
			ports = append(ports, uint16(p.PublishedPort))
		}
	}

	return hosts, ports, nil
}

func (d *docker) GetPorts() ([]uint16, error) {
	opts := types.ContainerListOptions{}
	containers, err := d.c.ContainerList(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	openPorts := []uint16{}
	for _, c := range containers {
		for _, p := range c.Ports {
			// When port is not published on the host docker return public port as 0, so we need to avoid it
			if p.PublicPort != 0 {
				openPorts = append(openPorts, p.PublicPort)
			}
		}
	}

	return openPorts, nil
}

func (d *docker) ContainerStats(name string) (io.ReadCloser, error) {
	stats, err := d.c.ContainerStats(context.Background(), name, true)
	return stats.Body, err
}

func (d *docker) ContainerResize(name string, rows, cols uint) error {
	return d.c.ContainerResize(context.Background(), name, types.ResizeOptions{Height: rows, Width: cols})
}

func (d *docker) ContainerRename(old, new string) error {
	return d.c.ContainerRename(context.Background(), old, new)
}

func (d *docker) CreateAttachConnection(name string) (net.Conn, error) {
	ctx := context.Background()

	conf := types.ContainerAttachOptions{true, true, true, true, "ctrl-^,ctrl-^", true}
	conn, err := d.c.ContainerAttach(ctx, name, conf)
	if err != nil {
		return nil, err
	}

	return conn.Conn, nil
}

func (d *docker) CopyToContainer(containerName, destination, fileName string, content io.Reader) error {
	contents, err := io.ReadAll(content)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	t := tar.NewWriter(&buf)
	if err := t.WriteHeader(&tar.Header{Name: fileName, Mode: 0600, Size: int64(len(contents)), ModTime: time.Now()}); err != nil {
		return err
	}

	if _, err := t.Write(contents); err != nil {
		return err
	}

	if err := t.Close(); err != nil {
		return err
	}

	return d.c.CopyToContainer(context.Background(), containerName, destination, &buf, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true, CopyUIDGID: true})
}

func (d *docker) CopyFromContainer(containerName, filePath string) (io.Reader, error) {
	rc, stat, err := d.c.CopyFromContainer(context.Background(), containerName, filePath)
	if err != nil {
		return nil, err
	}

	if stat.Mode.IsDir() {
		return nil, fmt.Errorf("Copying directories is not supported")
	}

	tr := tar.NewReader(rc)
	tr.Next()

	return tr, nil
}

func (d *docker) ContainerDelete(name string) error {
	err := d.c.ContainerRemove(context.Background(), name, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	d.c.VolumeRemove(context.Background(), name, true)

	return err
}

type CreateContainerOpts struct {
	Image          string
	SessionId      string
	ContainerName  string
	Hostname       string
	ServerCert     []byte
	ServerKey      []byte
	CACert         []byte
	Privileged     bool
	HostFQDN       string
	Labels         map[string]string
	Networks       []string
	NetAliases     []string
	DindVolumeSize string
	UserVolume     string
	LimitCPU       float64
	LimitMemory    int64
	Envs           []string
}

func (d *docker) ContainerCreate(opts CreateContainerOpts) (err error) {
	containerDir := "/opt/pwd"
	containerCertDir := fmt.Sprintf("%s/certs", containerDir)

	env := append(opts.Envs, fmt.Sprintf("SESSION_ID=%s", opts.SessionId))

	if len(opts.ServerCert) > 0 {
		env = append(env, `DOCKER_TLSCERT=\/opt\/pwd\/certs\/cert.pem`)
	}

	if len(opts.ServerKey) > 0 {
		env = append(env, `DOCKER_TLSKEY=\/opt\/pwd\/certs\/key.pem`)
	}

	if len(opts.CACert) > 0 {
		env = append(env, `DOCKER_TLSCACERT=\/opt\/pwd\/certs\/ca.pem`)
	}

	if len(opts.ServerCert) > 0 || len(opts.ServerKey) > 0 || len(opts.CACert) > 0 {
		env = append(env, "DOCKER_TLSENABLE=true")
	} else {
		env = append(env, "DOCKER_TLSENABLE=false")
	}

	h := &container.HostConfig{
		NetworkMode: container.NetworkMode(opts.SessionId),
		Privileged:  opts.Privileged,
		AutoRemove:  true,
		LogConfig:   container.LogConfig{Config: map[string]string{"max-size": "10m", "max-file": "1"}},
	}

	if config.DINDAppArmor != "" {
		h.SecurityOpt = []string{fmt.Sprintf("apparmor=%s", config.DINDAppArmor)}
	}

	if config.ExternalDindVolumeSize != "" {
		h.StorageOpt = map[string]string{"size": config.ExternalDindVolumeSize}
	}

	var pidsLimit = config.DefaultMaxPIDs
	h.Resources.PidsLimit = &pidsLimit

	memoryLimit := config.DefaultLimitMemory * Megabyte
	if opts.LimitMemory > 0 {
		if opts.LimitMemory > config.DefaultMaxMemory {
			opts.LimitMemory = config.DefaultMaxMemory
		}

		memoryLimit = opts.LimitMemory * Megabyte
	}

	if memoryLimit > 0 {
		h.Resources.Memory = memoryLimit
		log.Printf("Setting memory limit to %d MB for container %s\n", memoryLimit/Megabyte, opts.ContainerName)
	}

	cpuLimit := config.DefaultLimitCPUCore
	if opts.LimitCPU > 0 {
		if opts.LimitCPU > config.DefaultMaxCPUCore {
			opts.LimitCPU = config.DefaultMaxCPUCore
		}

		cpuLimit = opts.LimitCPU
	}

	if cpuLimit > 0 {
		h.Resources.NanoCPUs = int64(cpuLimit * 1e9)

		numCPUs := int(cpuLimit)
		if numCPUs > 0 {
			if numCPUs == 1 {
				h.Resources.CpusetCpus = "0"
			} else {
				h.Resources.CpusetCpus = fmt.Sprintf("0-%d", numCPUs-1)
			}

			log.Printf("Setting CPU limit to %.2f CPUs (cpuset: %s) for container %s\n", cpuLimit, h.Resources.CpusetCpus, opts.ContainerName)
		}
	}

	t := true
	h.Resources.OomKillDisable = &t

	env = append(env, fmt.Sprintf("PWD_HOST_FQDN=%s", opts.HostFQDN))
	cf := &container.Config{
		Hostname:     opts.Hostname,
		Image:        opts.Image,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          env,
		Labels:       opts.Labels,
	}

	networkConf := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			opts.Networks[0]: &network.EndpointSettings{
				Aliases: opts.NetAliases,
			},
		},
	}

	if config.ExternalDindVolume {
		_, err = d.c.VolumeCreate(context.Background(), volume.VolumeCreateBody{
			Driver: "xfsvol",
			DriverOpts: map[string]string{
				"size": opts.DindVolumeSize,
			},
			Name: opts.ContainerName,
		})

		if err != nil {
			return
		}

		h.Binds = []string{fmt.Sprintf("%s:/var/lib/docker", opts.ContainerName)}

		defer func() {
			if err != nil {
				d.c.VolumeRemove(context.Background(), opts.SessionId, true)
			}
		}()
	}

	if opts.UserVolume != "" {
		if h.Binds == nil {
			h.Binds = []string{}
		}

		h.Binds = append(h.Binds, fmt.Sprintf("%s:/data", opts.UserVolume))
	}

	search, err := d.c.ImageSearch(context.Background(), opts.Image, types.ImageSearchOptions{
		Limit: 5,
	})
	if err != nil {
		log.Printf("Error searching image %s: %v\n", opts.Image, err)

		log.Printf("Pulling image caused by search result not found [%s]\n", opts.Image)
		err = d.PullImage(context.Background(), opts.Image)
		if err != nil {
			return err
		}
	}

	if len(search) == 0 {
		log.Printf("Pulling image caused by image not found [%s]\n", opts.Image)
		err = d.PullImage(context.Background(), opts.Image)
		if err != nil {
			return err
		}
	} else if len(search) > 0 && config.AlwaysPull {
		log.Printf("Pulling image caused by always pull configuration [%s]\n", opts.Image)
		err = d.PullImage(context.Background(), opts.Image)
		if err != nil {
			return err
		}
	}

	container, err := d.c.ContainerCreate(context.Background(), cf, h, networkConf, opts.ContainerName)
	if err != nil {
		return err
	}

	// Connect remaining networks if there are any
	if len(opts.Networks) > 1 {
		for _, nid := range opts.Networks {
			err = d.c.NetworkConnect(context.Background(), nid, container.ID, &network.EndpointSettings{})
			if err != nil {
				return
			}
		}
	}

	if err = d.copyIfSet(opts.ServerCert, "cert.pem", containerCertDir, opts.ContainerName); err != nil {
		return
	}

	if err = d.copyIfSet(opts.ServerKey, "key.pem", containerCertDir, opts.ContainerName); err != nil {
		return
	}

	if err = d.copyIfSet(opts.CACert, "ca.pem", containerCertDir, opts.ContainerName); err != nil {
		return
	}

	err = d.c.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", opts.ContainerName, err)
		return
	}

	// Wait for container to be running before returning
	// This prevents race conditions with terminal resize operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// Timeout - Let's check what happened to the container
			cinfo, inspectErr := d.c.ContainerInspect(context.Background(), container.ID)
			if inspectErr != nil {
				log.Printf("Error: Container %s failed to start and cannot be inspected: %v\n", opts.ContainerName, inspectErr)
				return fmt.Errorf("container failed to start: %v", inspectErr)
			}

			log.Printf("Warning: Container %s not running after timeout. Status: %s, Error: %s\n", opts.ContainerName, cinfo.State.Status, cinfo.State.Error)

			if cinfo.State.ExitCode != 0 {
				log.Printf("Container %s exited with code %d\n", opts.ContainerName, cinfo.State.ExitCode)
			}

			return fmt.Errorf("container not running after timeout, status: %s", cinfo.State.Status)

		default:
			cinfo, inspectErr := d.c.ContainerInspect(context.Background(), container.ID)
			if inspectErr != nil {
				log.Printf("Error inspecting container %s: %v\n", opts.ContainerName, inspectErr)
				return fmt.Errorf("error inspecting container: %v", inspectErr)
			}

			if cinfo.State.Running {
				log.Printf("Container %s is now running\n", opts.ContainerName)
				return nil
			}

			// Check if container Exited / Crashed
			if cinfo.State.Status == "exited" || cinfo.State.Status == "dead" {
				log.Printf("Container %s failed to start. Status: %s, ExitCode: %d, Error: %s\n",
					opts.ContainerName, cinfo.State.Status, cinfo.State.ExitCode, cinfo.State.Error)
				return fmt.Errorf("container exited immediately: %s", cinfo.State.Error)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (d *docker) ContainerIPs(id string) (map[string]string, error) {
	cinfo, err := d.c.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}

	ips := map[string]string{}
	for networkId, conf := range cinfo.NetworkSettings.Networks {
		ips[networkId] = conf.IPAddress
	}

	return ips, nil
}

func (d *docker) PullImage(ctx context.Context, image string) error {
	_, err := reference.Parse(image)
	if err != nil {
		return err
	}

	options := types.ImageCreateOptions{}

	responseBody, err := d.c.ImageCreate(ctx, image, options)
	if err != nil {
		return err
	}

	_, err = io.Copy(ioutil.Discard, responseBody)

	return err
}

func (d *docker) copyIfSet(content []byte, fileName, path, containerName string) error {
	if len(content) > 0 {
		return d.CopyToContainer(containerName, path, fileName, bytes.NewReader(content))
	}

	return nil
}

func (d *docker) ExecAttach(instanceName string, command []string, out io.Writer) (int, error) {
	e, err := d.c.ContainerExecCreate(context.Background(), instanceName, types.ExecConfig{Cmd: command, AttachStdout: true, AttachStderr: true, Tty: true})
	if err != nil {
		return 0, err
	}

	resp, err := d.c.ContainerExecAttach(context.Background(), e.ID, types.ExecStartCheck{
		Tty: true,
	})

	if err != nil {
		return 0, err
	}

	io.Copy(out, resp.Reader)

	var ins types.ContainerExecInspect
	for _ = range time.Tick(1 * time.Second) {
		ins, err = d.c.ContainerExecInspect(context.Background(), e.ID)
		if ins.Running {
			continue
		}

		if err != nil {
			return 0, err
		}

		break
	}

	return ins.ExitCode, nil
}

func (d *docker) Exec(instanceName string, command []string) (int, error) {
	e, err := d.c.ContainerExecCreate(context.Background(), instanceName, types.ExecConfig{Cmd: command})
	if err != nil {
		return 0, err
	}

	err = d.c.ContainerExecStart(context.Background(), e.ID, types.ExecStartCheck{})
	if err != nil {
		return 0, err
	}

	var ins types.ContainerExecInspect
	for _ = range time.Tick(1 * time.Second) {
		ins, err = d.c.ContainerExecInspect(context.Background(), e.ID)
		if ins.Running {
			continue
		}

		if err != nil {
			return 0, err
		}

		break
	}

	return ins.ExitCode, nil
}

func (d *docker) NetworkDisconnect(containerId, networkId string) error {
	err := d.c.NetworkDisconnect(context.Background(), networkId, containerId, true)
	if err != nil {
		log.Printf("Disconnection of container from network err [%s]\n", err)
		return err
	}

	return nil
}

func (d *docker) NetworkDelete(id string) error {
	err := d.c.NetworkRemove(context.Background(), id)
	if err != nil {
		return err
	}

	return nil
}

func (d *docker) SwarmInit(advertiseAddr string) (*SwarmTokens, error) {
	req := swarm.InitRequest{AdvertiseAddr: advertiseAddr, ListenAddr: "0.0.0.0:2377"}
	_, err := d.c.SwarmInit(context.Background(), req)

	if err != nil {
		return nil, err
	}

	swarmInfo, err := d.c.SwarmInspect(context.Background())
	if err != nil {
		return nil, err
	}

	return &SwarmTokens{
		Worker:  swarmInfo.JoinTokens.Worker,
		Manager: swarmInfo.JoinTokens.Manager,
	}, nil
}

func (d *docker) SwarmJoin(addr, token string) error {
	req := swarm.JoinRequest{RemoteAddrs: []string{addr}, JoinToken: token, ListenAddr: "0.0.0.0:2377", AdvertiseAddr: "eth0"}
	return d.c.SwarmJoin(context.Background(), req)
}

func (d *docker) VolumeCreate(name string, labels map[string]string) error {
	_, err := d.c.VolumeCreate(context.Background(), volume.VolumeCreateBody{
		Name:   name,
		Labels: labels,
	})

	return err
}

func (d *docker) VolumeInspect(name string) (types.Volume, error) {
	return d.c.VolumeInspect(context.Background(), name)
}

func NewDocker(c *client.Client) *docker {
	return &docker{c: c}
}
