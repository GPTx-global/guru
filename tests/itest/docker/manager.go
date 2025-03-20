// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// Manager defines a docker pool instance, used to build, run, interact with and stop docker containers
// running Evmos nodes.
type Manager struct {
	pool     *dockertest.Pool
	Networks map[string]*dockertest.Network

	// Nodes stores the list of active containers in the network
	validatorNodes map[string][]*dockertest.Resource
	fullNodes      map[string][]*dockertest.Resource
	images         []string
}

// NewManager creates new docker pool and network and returns a populated Manager instance
func NewManager() (*Manager, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("docker pool creation error: %w", err)
	}

	return &Manager{
		pool:           pool,
		Networks:       make(map[string]*dockertest.Network),
		validatorNodes: make(map[string][]*dockertest.Resource),
		fullNodes:      make(map[string][]*dockertest.Resource),
	}, nil
}

// BuildImage builds a docker image to run in the provided context directory
// with <name>:<version> as the image target
func (m *Manager) BuildImage(name, version, dockerFile, contextDir string, args map[string]string) error {
	buildArgs := make([]docker.BuildArg, 0, len(args))
	for k, v := range args {
		bArg := docker.BuildArg{
			Name:  k,
			Value: v,
		}
		buildArgs = append(buildArgs, bArg)
	}
	opts := docker.BuildImageOptions{
		// local Dockerfile path
		Dockerfile: dockerFile,
		BuildArgs:  buildArgs,
		// rebuild the image every time in case there were changes
		// and the image is cached
		NoCache: true,
		// name with tag, e.g. guru:v0.5.0
		Name:         fmt.Sprintf("%s:%s", name, version),
		OutputStream: io.Discard,
		ErrorStream:  os.Stdout,
		ContextDir:   contextDir,
	}
	if err := m.Client().BuildImage(opts); err != nil {
		return err
	}
	m.images = append(m.images, opts.Name)
	return nil
}

func (m *Manager) CreateNetwork(name string) error {
	network, err := m.pool.CreateNetwork("guru-local-itest-" + name)
	if err != nil {
		return fmt.Errorf("docker network creation error: %w", err)
	}
	m.Networks[name] = network
	return nil
}

// RunNode creates a docker container from the provided node instance and runs it.
// To make sure the node started properly, get requests are sent to the JSON-RPC server repeatedly
// with a timeout of 60 seconds.
// In case the node fails to start, the container logs are returned along with the error.
func (m *Manager) RunNode(node *Node, val bool) error {
	var resource *dockertest.Resource
	var err error

	// Disabled temporal for error fixing
	if node.withRunOptions {
		resource, err = m.pool.RunWithOptions(node.RunOptions)

	} else {
		resource, err = m.pool.Run(node.repository, node.tag, []string{})
	}

	if err != nil && resource != nil {
		if resource == nil {
			return err
		}
		stdOut, stdErr, _ := m.GetLogs(resource.Container.ID)
		return fmt.Errorf(
			"can't run container\n\n[error stream]:\n\n%s\n\n[output stream]:\n\n%s",
			stdErr,
			stdOut,
		)
	}

	// trying to get JSON-RPC server, to make sure node started properly
	// the last returned error before deadline exceeded will be returned from .Retry()
	err = m.pool.Retry(
		func() error {
			// recreating container instance because resource.Container.State
			// does not update properly by default
			c, err := m.Client().InspectContainer(resource.Container.ID)
			if err != nil {
				return fmt.Errorf("can't inspect container: %s", err.Error())
			}
			// if node failed to start, i.e. ExitCode != 0, return container logs
			if c.State.ExitCode != 0 {
				stdOut, stdErr, _ := m.GetLogs(resource.Container.ID)
				return fmt.Errorf(
					"can't start evmos node, container exit code: %d\n\n[error stream]:\n\n%s\n\n[output stream]:\n\n%s",
					c.State.ExitCode,
					stdErr,
					stdOut,
				)
			}
			// get host:port for current container in local network
			addr := resource.GetHostPort(jrpcPort + "/tcp")
			r, err := http.Get("http://" + addr)
			if err != nil {
				return fmt.Errorf("can't get node json-rpc server: %s", err)
			}
			defer r.Body.Close()
			return nil
		},
	)

	if err != nil {
		stdOut, stdErr, _ := m.GetLogs(resource.Container.ID)
		return fmt.Errorf(
			"can't start node: %s\n\n[error stream]:\n\n%s\n\n[output stream]:\n\n%s",
			err.Error(),
			stdErr,
			stdOut,
		)
	}
	if val {
		m.validatorNodes[node.tag] = append(m.validatorNodes[node.tag], resource)
	} else {
		m.fullNodes[node.tag] = append(m.fullNodes[node.tag], resource)
	}

	return nil
}

// GetLogs returns the logs of the container with the provided containerID
func (m *Manager) GetLogs(id string) (stdOut, stdErr string, err error) {
	var outBuf, errBuf bytes.Buffer
	opts := docker.LogsOptions{
		Container:    id,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
		Stdout:       true,
		Stderr:       true,
	}
	err = m.Client().Logs(opts)
	if err != nil {
		return "", "", fmt.Errorf("can't get logs: %s", err)
	}
	return outBuf.String(), errBuf.String(), nil
}

func (m *Manager) WaitForNextBlock(ctx context.Context, id string) (string, error) {
	height, err := m.GetNodeHeight(ctx, id)
	if err != nil {
		return "", err
	}
	return m.WaitForHeight(ctx, id, height+1)
}

// WaitForHeight queries the Evmos node every second until the node will reach the specified height.
// After 5 minutes this function times out and returns an error
func (m *Manager) WaitForHeight(ctx context.Context, id string, height int) (string, error) {
	var currentHeight int
	var err error
	ticker := time.NewTicker(2 * time.Minute)
	for {
		select {
		case <-ticker.C:
			stdOut, stdErr, errLogs := m.GetLogs(id)
			if errLogs != nil {
				return "", fmt.Errorf("error while getting logs: %s", errLogs.Error())
			}
			return "", fmt.Errorf(
				"can't reach height %d, due to: %s\nerror logs: %s\nout logs: %s",
				height, err.Error(), stdOut, stdErr,
			)
		default:
			currentHeight, err = m.GetNodeHeight(ctx, id)
			if currentHeight >= height {
				if err != nil {
					return err.Error(), nil
				}
				return "", nil
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// GetNodeHeight calls the Evmos CLI in the current node container to get the current block height
func (m *Manager) GetNodeHeight(ctx context.Context, id string) (int, error) {
	exec, err := m.CreateExec([]string{"gurud", "q", "block"}, id)
	if err != nil {
		return 0, fmt.Errorf("create exec error: %w", err)
	}
	outBuff, errBuff, err := m.RunExec(ctx, exec)
	if err != nil {
		return 0, fmt.Errorf("run exec error: %w", err)
	}
	outStr := outBuff.String()
	var h int
	// parse current height number from block info
	if outStr != "<nil>" && outStr != "" {
		index := strings.Index(outBuff.String(), "\"height\":")
		qq := outStr[index+10 : index+12]
		h, err = strconv.Atoi(qq)
		// check if the conversion was possible
		if err == nil {
			// if conversion was possible but the errBuff is not empty, return the height along with an error
			// this is necessary e.g. when the "duplicate proto" errors occur in the logs but the node is still
			// producing blocks
			if errBuff.String() != "" {
				return h, fmt.Errorf("%s", errBuff.String())
			}
			return h, nil
		}
	}
	if errBuff.String() != "" {
		return 0, fmt.Errorf("guru query error: %s", errBuff.String())
	}
	return h, nil
}

// GetNodeVersion calls the Evmos CLI in the current node container to get the
// current node version
func (m *Manager) GetNodeVersion(ctx context.Context, id string) (string, error) {
	exec, err := m.CreateExec([]string{"gurud", "version"}, id)
	if err != nil {
		return "", fmt.Errorf("create exec error: %w", err)
	}
	outBuff, errBuff, err := m.RunExec(ctx, exec)
	if err != nil {
		return "", fmt.Errorf("run exec error: %w", err)
	}
	if errBuff.String() != "" {
		return "", fmt.Errorf("guru version error: %s", errBuff.String())
	}
	return outBuff.String(), nil
}

// GetNodeId calls the Evmos CLI in the current node container to get the
// current node id
func (m *Manager) GetNodeId(ctx context.Context, id string) (string, error) {
	exec, err := m.CreateExec([]string{"gurud", "status"}, id)
	if err != nil {
		return "", fmt.Errorf("create exec error: %w", err)
	}
	outBuff, errBuff, err := m.RunExec(ctx, exec)
	if err != nil {
		return "", fmt.Errorf("run exec error: %w", err)
	}
	outStr := outBuff.String()
	// parse node id from node status info
	if outStr != "<nil>" && outStr != "" {
		index := strings.Index(outBuff.String(), "\"id\":")
		qq := outStr[index+6 : index+46]
		return qq, nil
	}
	if errBuff.String() != "" {
		return "", fmt.Errorf("guru query error: %s", errBuff.String())
	}
	return "", nil
}

func (m *Manager) GetContainerId(network string, index int, getFullnode bool) (string, error) {
	nodes := m.validatorNodes[network]
	if getFullnode {
		nodes = m.fullNodes[network]
	}

	if len(nodes) <= index {
		return "", fmt.Errorf("container node with index %d is not found", index)
	}
	return nodes[index].Container.ID, nil

}

func (m *Manager) GetValidatorCount(network string) int {
	return len(m.validatorNodes[network])
}

func (m *Manager) GetFullnodeCount(network string) int {
	return len(m.fullNodes[network])
}

func (m *Manager) CopyFileFromContainer(ctx context.Context, containerId, containerPath, localPath string) error {
	// Create a temporary file to store the tar data
	tmpFile, err := os.CreateTemp("", "docker-file-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file
	defer tmpFile.Close()

	// Get the file from the container as a tar archive
	err = m.Client().DownloadFromContainer(containerId, docker.DownloadFromContainerOptions{
		Context:           ctx,
		Path:              containerPath,
		OutputStream:      tmpFile,
		InactivityTimeout: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to download from container: %w", err)
	}

	// Rewind file for reading
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to rewind temp file: %w", err)
	}

	// Extract actual file from tar
	return m.extractTarFile(tmpFile, localPath)

	// return nil
}

// The Client method returns the Docker client used by the Manager's pool
func (m *Manager) Client() *docker.Client {
	return m.pool.Client
}

// RemoveNetwork removes the Manager's used network from the pool
func (m *Manager) RemoveNetwork(name string) error {
	return m.pool.RemoveNetwork(m.Networks[name])
}

func (m *Manager) RemoveAllNetworks() error {
	for name := range m.Networks {
		if err := m.RemoveNetwork(name); err != nil {
			return err
		}
	}
	return nil
}

// KillNode stops the execution of the specific container
func (m *Manager) KillNode(id string) error {
	return m.pool.Client.StopContainer(id, 5)
}

func (m *Manager) RestartNode(id string) error {
	return m.pool.Client.RestartContainer(id, 5)
}

// RemoveNode removes the container
func (m *Manager) RemoveNode(id string) error {
	options := &docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
		Context:       context.Background(),
	}
	if err := m.pool.Client.RemoveContainer(*options); err != nil {
		return err
	}
	return nil
}

func (m *Manager) KillAllNodesByNetwork(network string) error {
	for _, node := range m.validatorNodes[network] {
		if err := m.KillNode(node.Container.ID); err != nil {
			return err
		}
	}
	for _, node := range m.fullNodes[network] {
		if err := m.KillNode(node.Container.ID); err != nil {
			return err
		}
	}
	return nil
}

// KillAllNodes stops the execution of all running containers
func (m *Manager) KillAllNodes() error {
	for network := range m.Networks {
		if err := m.KillAllNodesByNetwork(network); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) RemoveAllNodesByNetwork(network string) error {
	for _, node := range m.validatorNodes[network] {
		if err := m.RemoveNode(node.Container.ID); err != nil {
			return err
		}
	}
	for _, node := range m.fullNodes[network] {
		if err := m.RemoveNode(node.Container.ID); err != nil {
			return err
		}
	}
	return nil
}

// RemoveAllNodes removes all containers
func (m *Manager) RemoveAllNodes() error {
	for network := range m.Networks {
		if err := m.RemoveAllNodesByNetwork(network); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) RemoveImage(image string) error {
	err := m.pool.Client.RemoveImage(image)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) RemoveAllImages() error {
	for _, image := range m.images {
		if err := m.RemoveImage(image); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ClearNetwork(name string) error {
	if err := m.KillAllNodesByNetwork(name); err != nil {
		return err
	}
	if err := m.RemoveAllNodesByNetwork(name); err != nil {
		return err
	}
	m.validatorNodes[name] = []*dockertest.Resource{}
	m.fullNodes[name] = []*dockertest.Resource{}
	// if err := m.RemoveImage(name); err != nil {
	// 	return err
	// }
	return m.RemoveNetwork(name)
}

func (m *Manager) Clear() error {
	if err := m.KillAllNodes(); err != nil {
		return err
	}
	if err := m.RemoveAllNodes(); err != nil {
		return err
	}
	if err := m.RemoveAllNetworks(); err != nil {
		return err
	}
	return nil
	// return m.RemoveAllImages()
}

// Extracts a file from a tar archive and saves it to disk
func (m *Manager) extractTarFile(tarFile *os.File, destPath string) error {
	// Open tar reader
	tr := tar.NewReader(tarFile)

	// Extract the first file found
	_, err := tr.Next()
	if err == io.EOF {
		return fmt.Errorf("no files found in tar archive")
	}
	if err != nil {
		return fmt.Errorf("failed to read tar file: %w", err)
	}

	// Create destination file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create extracted file: %w", err)
	}
	defer outFile.Close()

	// Copy tar file content to new file
	if _, err := io.Copy(outFile, tr); err != nil {
		return fmt.Errorf("failed to write extracted file: %w", err)
	}

	fmt.Printf("✅ File updated: %s\n", destPath)
	return nil

}
