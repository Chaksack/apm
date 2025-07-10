// Copyright (c) 2024 APM Solution Contributors
// Authors: Andrew Chakdahah (chakdahah@gmail.com) and Yaw Boateng Kessie (ybkess@gmail.com)
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

package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// startTime is used to calculate uptime
var startTime = time.Now()

// SystemStatus represents the APM system status
type SystemStatus struct {
	Status     string                 `json:"status"`
	Version    string                 `json:"version"`
	Uptime     string                 `json:"uptime"`
	Components map[string]string      `json:"components"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Status returns the APM system status
func Status(c *fiber.Ctx) error {
	uptime := time.Since(startTime).Round(time.Second).String()

	status := SystemStatus{
		Status:  "operational",
		Version: "1.0.0",
		Uptime:  uptime,
		Components: map[string]string{
			"prometheus":   "healthy",
			"grafana":      "healthy",
			"alertmanager": "healthy",
			"loki":         "healthy",
			"promtail":     "healthy",
		},
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"timezone":  time.Local.String(),
		},
	}

	return c.JSON(status)
}
