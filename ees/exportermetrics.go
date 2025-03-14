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

package ees

import (
	"sync"
	"time"

	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/cron"
)

// exporterMetrics stores export statistics with thread-safe access and
// cron-scheduled resets.
type exporterMetrics struct {
	mu   sync.RWMutex
	ms   utils.MapStorage
	cron *cron.Cron
	loc  *time.Location
}

// newExporterMetrics creates metrics with optional automatic reset.
// schedule is a cron expression for reset timing (empty to disable).
func newExporterMetrics(schedule string, loc *time.Location) *exporterMetrics {
	m := &exporterMetrics{
		loc: loc,
	}
	m.Reset() // init MapStorage with default values

	if schedule != "" {
		m.cron = cron.New()
		m.cron.AddFunc(schedule, func() {
			m.Reset()
		})
		m.cron.Start()
	}
	return m
}

// Reset immediately clears all metrics and resets them to initial values.
func (m *exporterMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ms = utils.MapStorage{
		utils.NumberOfEvents:  int64(0),
		utils.PositiveExports: utils.StringSet{},
		utils.NegativeExports: utils.StringSet{},
		utils.TimeNow:         time.Now().In(m.loc),
	}
}

// StopCron stops the automatic reset schedule if one is active.
func (m *exporterMetrics) StopCron() {
	if m.cron == nil {
		return
	}
	m.cron.Stop()
	// ctx := m.cron.Stop()
	// <-ctx.Done() // wait for any running jobs to complete
}

// String returns the map as json string.
func (m *exporterMetrics) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ms.String()
}

// FieldAsInterface returns the value from the path.
func (m *exporterMetrics) FieldAsInterface(fldPath []string) (val any, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ms.FieldAsInterface(fldPath)
}

// FieldAsString returns the value from path as string.
func (m *exporterMetrics) FieldAsString(fldPath []string) (str string, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ms.FieldAsString(fldPath)
}

// Set sets the value at the given path.
func (m *exporterMetrics) Set(fldPath []string, val any) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ms.Set(fldPath, val)
}

// GetKeys returns all the keys from map.
func (m *exporterMetrics) GetKeys(nested bool, nestedLimit int, prefix string) (keys []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ms.GetKeys(nested, nestedLimit, prefix)
}

// Remove removes the item at path.
func (m *exporterMetrics) Remove(fldPath []string) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ms.Remove(fldPath)
}

func (m *exporterMetrics) ClonedMapStorage() (msClone utils.MapStorage) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ms.Clone()
}

// IncrementEvents increases the event counter (NumberOfEvents) by 1.
func (m *exporterMetrics) IncrementEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	count, _ := m.ms[utils.NumberOfEvents].(int64)
	m.ms[utils.NumberOfEvents] = count + 1
}

func (m *exporterMetrics) updateMetrics(cgrID string, ev engine.MapEvent, hasError bool, timezone string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if hasError {
		m.ms[utils.NegativeExports].(utils.StringSet).Add(cgrID)
	} else {
		m.ms[utils.PositiveExports].(utils.StringSet).Add(cgrID)
	}
	if aTime, err := ev.GetTime(utils.AnswerTime, timezone); err == nil {
		if _, has := m.ms[utils.FirstEventATime]; !has {
			m.ms[utils.FirstEventATime] = time.Time{}
		}
		if _, has := m.ms[utils.LastEventATime]; !has {
			m.ms[utils.LastEventATime] = time.Time{}
		}
		if m.ms[utils.FirstEventATime].(time.Time).IsZero() ||
			aTime.Before(m.ms[utils.FirstEventATime].(time.Time)) {
			m.ms[utils.FirstEventATime] = aTime
		}
		if aTime.After(m.ms[utils.LastEventATime].(time.Time)) {
			m.ms[utils.LastEventATime] = aTime
		}
	}
	if oID, err := ev.GetTInt64(utils.OrderID); err == nil {
		if _, has := m.ms[utils.FirstExpOrderID]; !has {
			m.ms[utils.FirstExpOrderID] = int64(0)
		}
		if _, has := m.ms[utils.LastExpOrderID]; !has {
			m.ms[utils.LastExpOrderID] = int64(0)
		}
		if m.ms[utils.FirstExpOrderID].(int64) == 0 ||
			m.ms[utils.FirstExpOrderID].(int64) > oID {
			m.ms[utils.FirstExpOrderID] = oID
		}
		if m.ms[utils.LastExpOrderID].(int64) < oID {
			m.ms[utils.LastExpOrderID] = oID
		}
	}
	if cost, err := ev.GetFloat64(utils.Cost); err == nil {
		if _, has := m.ms[utils.TotalCost]; !has {
			m.ms[utils.TotalCost] = float64(0.0)
		}
		m.ms[utils.TotalCost] = m.ms[utils.TotalCost].(float64) + cost
	}
	if tor, err := ev.GetString(utils.ToR); err == nil {
		if usage, err := ev.GetDuration(utils.Usage); err == nil {
			switch tor {
			case utils.MetaVoice:
				if _, has := m.ms[utils.TotalDuration]; !has {
					m.ms[utils.TotalDuration] = time.Duration(0)
				}
				m.ms[utils.TotalDuration] = m.ms[utils.TotalDuration].(time.Duration) + usage
			case utils.MetaSMS:
				if _, has := m.ms[utils.TotalSMSUsage]; !has {
					m.ms[utils.TotalSMSUsage] = time.Duration(0)
				}
				m.ms[utils.TotalSMSUsage] = m.ms[utils.TotalSMSUsage].(time.Duration) + usage
			case utils.MetaMMS:
				if _, has := m.ms[utils.TotalMMSUsage]; !has {
					m.ms[utils.TotalMMSUsage] = time.Duration(0)
				}
				m.ms[utils.TotalMMSUsage] = m.ms[utils.TotalMMSUsage].(time.Duration) + usage
			case utils.MetaGeneric:
				if _, has := m.ms[utils.TotalGenericUsage]; !has {
					m.ms[utils.TotalGenericUsage] = time.Duration(0)
				}
				m.ms[utils.TotalGenericUsage] = m.ms[utils.TotalGenericUsage].(time.Duration) + usage
			case utils.MetaData:
				if _, has := m.ms[utils.TotalDataUsage]; !has {
					m.ms[utils.TotalDataUsage] = time.Duration(0)
				}
				m.ms[utils.TotalDataUsage] = m.ms[utils.TotalDataUsage].(time.Duration) + usage
			}
		}
	}
}
