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
	"context"

	dc "github.com/docker/docker/client"

	"github.com/sealerio/sealer/pkg/infra/container/client"
)

type Provider struct {
	DockerClient *dc.Client
	Ctx          context.Context
}

/* 
	创建一个新的docker服务者
	工厂函数:封装了 docker 客户端以及上下文对象
*/
func NewDockerProvider() (client.ProviderService, error) {
	// 创建一个新的上下文对象 ctx
	ctx := context.Background()
	// 创建一个新的 docker 客户端 并 存储到 cli
	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Provider{
		Ctx:          ctx,
		DockerClient: cli,
	}, nil
}
