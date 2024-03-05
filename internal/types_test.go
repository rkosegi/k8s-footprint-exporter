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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadNonExistentMetric(t *testing.T) {
	mc := &MetricConfig{}
	assert.Error(t, mc.LoadFrom("this file does not exists"))
}

func TestLoadMetricFromInvalidFile(t *testing.T) {
	mc := &MetricConfig{}
	assert.Error(t, mc.LoadFrom("../testdata/not_a_yaml_file.yaml"))
}

func TestLoadMetric(t *testing.T) {
	mc := &MetricConfig{}
	err := mc.LoadFrom("../testdata/metrics1.yaml")
	assert.NoError(t, err)
}

func TestLoadInvalidMetric(t *testing.T) {
	mc := &MetricConfig{}
	err := mc.LoadFrom("../testdata/metrics2.yaml")
	assert.Error(t, err)
}

func TestLoadMetricFailNormalize(t *testing.T) {
	mc := &MetricConfig{}
	err := mc.LoadFrom("../testdata/metrics3.yaml")
	assert.Error(t, err)
}
