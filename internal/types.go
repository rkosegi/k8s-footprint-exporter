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
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	namespace = "k8sfootprint"
	exporter  = "exporter"
	bTrue     = true
	bFalse    = false
)

type MatchingFn func(string) bool

func MatchAllNames(string) bool {
	return true
}

func MatchRe(re *regexp.Regexp) MatchingFn {
	return func(name string) bool {
		return re.MatchString(name)
	}
}

type CollectorOpts struct {
	Cfg           *MetricConfig
	Namespace     string
	DynamicClient *dynamic.DynamicClient
	Client        *clientset.Clientset
	Log           *slog.Logger
}

type MetricSet struct {
	// Flag indicating whether to group resource size by name or not.
	// When true, every resource will have its own metric with label "resource_name" set to actual name
	NameLabel *bool `yaml:"nameLabel,omitempty"`
	// Regular expression to filter resources
	IncludeOnly *regexp.Regexp `yaml:"includeOnly,omitempty"`
	// Estimated size of resource serialized as JSON
	Size *bool `yaml:"size,omitempty"`
	// Number of resources in this metric set
	Count *bool `yaml:"count,omitempty"`
	// not persisted fields
	// resolved MatchingFn
	ResourceNameMatcher MatchingFn `yaml:"-"`
	// v1.APIResource associated with this metric set
	Schema *v1.APIResource `yaml:"-"`
	// schema.GroupVersion for this metric set
	GV *schema.GroupVersion `yaml:"-"`
}

// Normalize provides default values for nil fields and ensures that structure is valid.
func (ms *MetricSet) Normalize() error {
	if ms.NameLabel == nil {
		ms.NameLabel = &bFalse
	}
	if ms.Count == nil {
		ms.Count = &bTrue
	}
	if ms.Size == nil {
		ms.Size = &bTrue
	}
	if ms.IncludeOnly == nil {
		ms.ResourceNameMatcher = MatchAllNames
	} else {
		ms.ResourceNameMatcher = MatchRe(ms.IncludeOnly)
	}
	if !*ms.Count && !*ms.Size {
		return errors.New("neither count nor size metric are enabled")
	}
	return nil
}

type ResourceSet struct {
	APIVersion string                `yaml:"apiVersion"`
	Kinds      map[string]*MetricSet `yaml:"kinds"`
}

type MetricConfig map[string]*ResourceSet

func (mc *MetricConfig) LoadFrom(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, mc)
	if err != nil {
		return err
	}
	for _, rs := range *mc {
		for kind, ms := range rs.Kinds {
			errs := validation.IsDNS1123Label(kind)
			if len(errs) > 0 {
				return errors.New(strings.Join(errs, ","))
			}
			if ms == nil {
				ms = &MetricSet{}
				rs.Kinds[kind] = ms
			}
			if err = ms.Normalize(); err != nil {
				return err
			}
		}
	}
	return nil
}
