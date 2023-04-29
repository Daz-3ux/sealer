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

package main

import (
	"github.com/containers/buildah"

	"github.com/sealerio/sealer/cmd/sealer/boot"
	"github.com/sealerio/sealer/cmd/sealer/cmd"
)

func main() {
	/*
		检测当前进程是否是一个重新执行的子进程
	*/
	if buildah.InitReexec() {
		// 如果是重新执行的子进程，立刻返回
		return
	}
	/*
		执行初始化操作
	*/
	if err := boot.OnBoot(); err != nil {
		panic(err)
	}
	/*
		如果一切正常,开始执行程序的主要逻辑
	*/
	cmd.Execute()
}
