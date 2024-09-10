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

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type collector struct {
	opts      *CollectorOpts
	scrapeSum prometheus.Summary
	error     prometheus.Gauge
	up        prometheus.Gauge

	schema struct {
		lock sync.Mutex
		arls []*v1.APIResourceList
	}
}

func (c *collector) scrape(ch chan<- prometheus.Metric) {
	size, count := c.makeMetrics()

	errCnt := 0
	defer func(start time.Time) {
		if errCnt > 0 {
			c.error.Add(float64(errCnt))
			c.up.Set(0)
		} else {
			c.up.Set(1)
		}
		c.scrapeSum.Observe(time.Since(start).Seconds())
	}(time.Now())

	for rsname, rs := range *c.opts.Cfg {
		for kind, ms := range rs.Kinds {
			err := c.ensureSchema(rs.APIVersion, kind, ms)
			if err != nil {
				c.opts.Log.Warn("Can't resolve schema, maybe some CRDs are missing?",
					"apiVersion", rs.APIVersion, "kind", kind, "err", err)
				errCnt++
				continue
			}

			items, err := c.fetchList(ms.Schema, *ms.GV, kind)
			if err != nil {
				c.opts.Log.Warn("Failed to list resources", "gv", ms.GV.String(), "err", err)
				errCnt++
				continue
			}
			err = c.appendMetric(rsname, items, ms, size, count, ms.GV.String(), kind)
			if err != nil {
				c.opts.Log.Warn("Failed extract metric", "gv", ms.GV.String(), "err", err)
				errCnt++
				continue
			}
		}
	}
	size.Collect(ch)
	count.Collect(ch)
}

func (c *collector) makeMetrics() (*prometheus.GaugeVec, *prometheus.GaugeVec) {
	size := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "resources",
		Name:      "size",
		Help:      "Estimated size of resources in this resource set, serialized as JSON",
	}, []string{"resource_set", "resource_name", "apiVersion", "kind"})
	count := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "resources",
		Name:      "count",
		Help:      "Total number of resources in this resource set",
	}, []string{"resource_set", "resource_name", "apiVersion", "kind"})
	return size, count
}

func (c *collector) appendMetric(rsname string, items *unstructured.UnstructuredList, ms *MetricSet,
	size *prometheus.GaugeVec, count *prometheus.GaugeVec, gv string, kind string) (err error) {
	var matched []unstructured.Unstructured
	for _, item := range items.Items {
		if ms.ResourceNameMatcher(item.GetName()) {
			matched = append(matched, item)
		}
	}
	if len(items.Items) == 0 {
		return nil
	}

	if *ms.Size {
		if *ms.NameLabel {
			err = c.appendPerItemSize(rsname, matched, size, gv, kind)
		} else {
			err = c.appendSumSize(rsname, matched, size, gv, kind)
		}
		if err != nil {
			return err
		}
	}
	if *ms.Count {
		count.WithLabelValues(rsname, "*", gv, kind).Set(float64(len(matched)))
	}
	return nil
}

func (c *collector) appendPerItemSize(rsname string, items []unstructured.Unstructured,
	size *prometheus.GaugeVec, gv string, kind string) error {
	for _, item := range items {
		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		size.WithLabelValues(rsname, item.GetName(), gv, kind).Set(float64(len(data)))
	}
	return nil
}

func (c *collector) appendSumSize(rsname string, items []unstructured.Unstructured,
	size *prometheus.GaugeVec, gv string, kind string) error {
	sum := 0
	for _, item := range items {
		data, err := item.MarshalJSON()
		if err != nil {
			return err
		}
		sum += len(data)
	}
	size.WithLabelValues(rsname, "*", gv, kind).Set(float64(sum))
	return nil
}

func (c *collector) Describe(_ chan<- *prometheus.Desc) {}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.scrape(ch)
	ch <- c.scrapeSum
	ch <- c.error
	ch <- c.up
}

func NewCollector(opts *CollectorOpts) prometheus.Collector {
	c := &collector{
		opts: opts,
	}
	c.setup()
	return c
}
