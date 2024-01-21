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
	"context"
	"fmt"

	"github.com/samber/lo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func (c *collector) fetchSchema() error {
	_, arl, err := c.opts.Client.Discovery().ServerGroupsAndResources()
	if err != nil {
		return err
	}
	c.schema.arls = arl
	return nil
}

func (c *collector) ensureSchema(apiVersion string, kind string, ms *MetricSet) error {
	if ms.GV == nil {
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			return err
		}
		ms.GV = &gv
	}
	if ms.Schema == nil {
		ar, err := c.getSchemaFor(*ms.GV, kind)
		if err != nil {
			return err
		}
		ms.Schema = ar
		if ar == nil {
			return fmt.Errorf("no schema for %s:%s", ms.GV.String(), kind)
		}
	}
	return nil
}

func (c *collector) fetchList(ar *v1.APIResource, gv schema.GroupVersion, kind string) (*unstructured.UnstructuredList, error) {
	var rsrs dynamic.ResourceInterface
	rsrs = c.opts.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: kind,
	})
	if ar.Namespaced {
		rsrs = rsrs.(dynamic.NamespaceableResourceInterface).Namespace(c.opts.Namespace)
	}
	return rsrs.List(context.TODO(), v1.ListOptions{})
}

func (c *collector) getSchemaFor(gv schema.GroupVersion, kind string) (*v1.APIResource, error) {
	c.schema.lock.Lock()
	defer c.schema.lock.Unlock()

	if c.schema.arls == nil {
		if err := c.fetchSchema(); err != nil {
			return nil, err
		}
	}
	arl, found := lo.Find(c.schema.arls, func(item *v1.APIResourceList) bool {
		return item.GroupVersion == gv.String()
	})
	if !found {
		return nil, nil
	}
	if ar, found := lo.Find(arl.APIResources, func(item v1.APIResource) bool {
		return item.Name == kind
	}); found {
		return &ar, nil
	}
	return nil, nil
}
