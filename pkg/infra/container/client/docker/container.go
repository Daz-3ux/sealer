// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/pkg/infra/container/client"
)

const (
	CgroupNoneDriver = "none"
)

// 获取 docker 主机的用户命名空间模式: host/private/shared/slave/""
func (p *Provider) getUserNsMode() (container.UsernsMode, error) {
	sysInfo, err := p.DockerClient.Info(p.Ctx)
	if err != nil {
		return "", err
	}

	var usernsMode container.UsernsMode
	for _, opt := range sysInfo.SecurityOptions {
		if opt == "name=userns" {
			// 采用 host 模式:不启用用户命名空间隔离
			usernsMode = "host"
		}
	}
	return usernsMode, err
}

// 设置容器挂载点
func (p *Provider) setContainerMount(opts *client.CreateOptsForContainer) []mount.Mount {
	mounts := DefaultMounts()
	if opts.Mount != nil {
		mounts = append(mounts, opts.Mount...)
	}
	return mounts
}

// 创建并运行一个 docker 容器
func (p *Provider) RunContainer(opts *client.CreateOptsForContainer) (string, error) {
	//docker run --hostname master1 --name master1
	//--privileged
	//--security-opt seccomp=unconfined --security-opt apparmor=unconfined
	//--tmpfs /tmp --tmpfs /run
	//--volume /var --volume /lib/modules:/lib/modules:ro
	//--device /dev/fuse
	//--detach --tty --restart=on-failure:1 --init=false sealer-io/sealer-base-image:latest

	// 准备网络资源
	networkID, err := p.PrepareNetworkResource(opts.NetworkName)
	if err != nil {
		return "", err
	}

	// 拉取镜像
	_, err = p.PullImage(opts.ImageName)
	if err != nil {
		return "", err
	}

	// 获取 docker 主机的用户命名空间模式
	mod, _ := p.getUserNsMode()
	// 设置容器挂载点
	mounts := p.setContainerMount(opts)
	falseOpts := false
	// 创建配置对象
																										// 容器配置对象
	resp, err := p.DockerClient.ContainerCreate(p.Ctx, &container.Config{
		Image:        opts.ImageName,
		Tty:          true,
		Labels:       opts.ContainerLabel,
		Hostname:     opts.ContainerHostName,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
	},
		// 主机配置对象
		&container.HostConfig{
			// 设置容器用户命名空间模式
			UsernsMode: mod,
			// 禁用 seccomp & apparmor
			SecurityOpt: []string{
				"seccomp=unconfined", "apparmor=unconfined",
			},
			// 设置重启策略
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 1,
			},
			// 不启用主进程
			Init:         &falseOpts,
			// 使用主机的 cgroup,共享资源
			CgroupnsMode: "host",
			// 特权模式开
			Privileged:   true,
			// 设置挂载点
			Mounts:       mounts,
		}, 
			// 网络配置对象
			&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				opts.NetworkName: {
					NetworkID: networkID,
				},
			},
		}, nil, opts.ContainerName)

	if err != nil {
		return "", err
	}

	// 启动容器
	err = p.DockerClient.ContainerStart(p.Ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}
	logrus.Infof("create container %s successfully", opts.ContainerName)
	return resp.ID, nil
}

// 获取容器信息
func (p *Provider) GetContainerInfo(containerID string, networkName string) (*client.Container, error) {
	resp, err := p.DockerClient.ContainerInspect(p.Ctx, containerID)
	if err != nil {
		return nil, err
	}
	return &client.Container{
		ContainerName:     resp.Name,
		ContainerIP:       resp.NetworkSettings.Networks[networkName].IPAddress,
		ContainerHostName: resp.Config.Hostname,
		ContainerLabel:    resp.Config.Labels,
		Status:            resp.State.Status,
	}, nil
}

//根据容器 IP 地址以及网络名称获取容器 ID 
func (p *Provider) GetContainerIDByIP(containerIP string, networkName string) (string, error) {
	// resp 存储 docker 中所有的容器
	resp, err := p.DockerClient.ContainerList(p.Ctx, types.ContainerListOptions{})
	if err != nil {
		return "", err
	}

	// 遍历每一个现有容器
	for _, item := range resp {
		if net, ok := item.NetworkSettings.Networks[networkName]; ok {
			if containerIP == net.IPAddress {
				return item.ID, nil
			}
		}
	}
	return "", err
}

// 删除容器
func (p *Provider) RmContainer(containerID string) error {
	err := p.DockerClient.ContainerRemove(p.Ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})

	if err != nil {
		return err
	}

	logrus.Infof("delete container %s successfully", containerID)
	return nil
}

// 获取 docker 服务器的(主机)系统信息
func (p *Provider) GetServerInfo() (*client.DockerInfo, error) {
	sysInfo, err := p.DockerClient.Info(p.Ctx)
	if err != nil {
		return nil, err
	}

	var dInfo client.DockerInfo

	// When CgroupDriver == "none", the MemoryLimit/PidsLimit/CPUShares
	// values are meaningless and need to be considered false.
	// https://github.com/moby/moby/issues/42151
	dInfo.CgroupVersion = sysInfo.CgroupVersion
	dInfo.StorageDriver = sysInfo.Driver
	dInfo.SecurityOptions = sysInfo.SecurityOptions
	dInfo.CgroupDriver = sysInfo.CgroupDriver
	if sysInfo.CgroupDriver == CgroupNoneDriver {
		return &dInfo, nil
	}
	dInfo.MemoryLimit = sysInfo.MemoryLimit
	dInfo.PidsLimit = sysInfo.PidsLimit
	dInfo.CPUShares = sysInfo.CPUShares
	dInfo.CPUNumber = sysInfo.NCPU
	return &dInfo, nil
}
