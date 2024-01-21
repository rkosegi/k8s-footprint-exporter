// Copyright 2024 Richard Kosegi
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

package internal

import "github.com/prometheus/client_golang/prometheus"

func (c *collector) setup() {
	c.scrapeSum = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: namespace,
		Subsystem: exporter,
		Name:      "last_scrape",
		Help:      "Summary of the last scrape of metrics from K8s.",
	})
	c.error = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: exporter,
		Name:      "last_scrape_error",
		Help:      "Whether the last scrape of metrics from K8s resulted in an error.",
	})
	c.up = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "up",
		Help:      "Whether the exporter is considered up.",
	})
}
