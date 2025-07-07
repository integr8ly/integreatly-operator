/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
	"time"
)

const (
	defaultRequeueTime = 10 * time.Second
)

type ControllerOptions struct {
	AddonInstanceNamespace string
	AddonInstanceName      string
	HeartBeatInterval      time.Duration
}

type ControllerConfig interface {
	ConfigureStatusController(*ControllerOptions)
}

func (c *ControllerOptions) Option(opts ...ControllerConfig) {
	for _, opt := range opts {
		opt.ConfigureStatusController(c)
	}
}

func (c *ControllerOptions) Default() {
	if c.HeartBeatInterval == 0 {
		c.HeartBeatInterval = defaultRequeueTime
	}
}

type WithAddonInstanceNamespace string

func (w WithAddonInstanceNamespace) ConfigureStatusController(c *ControllerOptions) {
	c.AddonInstanceNamespace = string(w)
}

type WithAddonInstanceName string

func (w WithAddonInstanceName) ConfigureStatusController(c *ControllerOptions) {
	c.AddonInstanceName = string(w)
}

type WithHeartbeatInterval time.Duration

func (w WithHeartbeatInterval) ConfigureStatusController(c *ControllerOptions) {
	c.HeartBeatInterval = time.Duration(w)
}
