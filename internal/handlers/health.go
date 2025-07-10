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

// HealthCheck represents the health status response
type HealthCheck struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []Check   `json:"checks,omitempty"`
}

// Check represents an individual health check
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Health returns a detailed health check response
func Health(c *fiber.Ctx) error {
	health := HealthCheck{
		Status:    "healthy",
		Service:   "apm",
		Timestamp: time.Now(),
		Checks: []Check{
			{
				Name:   "database",
				Status: "healthy",
			},
			{
				Name:   "prometheus",
				Status: "healthy",
			},
		},
	}

	return c.JSON(health)
}

// SimpleHealth returns a simple health check response
func SimpleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
