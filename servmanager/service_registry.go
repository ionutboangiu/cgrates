/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package servmanager

import (
	"sync"

	"github.com/cgrates/cgrates/utils"
)

// ServiceRegistry provides concurrent-safe registration and lookup of Service instances
// indexed by their unique names. The Service instances are stored in an ordered map.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services *utils.OrderedMap[string, Service]
}

// NewServiceRegistry returns an initialized registry for managing services.
// The registry is safe for concurrent access.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: utils.NewOrderedMap[string, Service](),
	}
}

// Lookup returns the Service for id or nil if not found. Safe for concurrent use.
func (r *ServiceRegistry) Lookup(id string) Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, _ := r.services.Get(id)
	return svc
}

// Register adds or updates Services using their name as the unique identifier.
// Will overwrite existing services if name conflicts.
func (r *ServiceRegistry) Register(svcs ...Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, svc := range svcs {
		r.services.Set(svc.ServiceName(), svc)
	}
}

// Unregister removes Services by ID.
func (r *ServiceRegistry) Unregister(ids ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		r.services.Delete(id)
	}
}

// List returns a new slice containing all registered Services.
// Order is not guaranteed.
func (r *ServiceRegistry) List() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services.Values()
}
