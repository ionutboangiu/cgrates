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

package config

import (
	"slices"
	"time"

	"github.com/cgrates/cgrates/utils"
)

// CoreSCfg the config for the coreS
type CoreSCfg struct {
	Caps                    int
	CapsStrategy            string
	CapsStatsInterval       time.Duration
	ShutdownTimeout         time.Duration
	InternalMetricsInterval time.Duration
	EEsConns                []string
	StatSConns              []string
}

func (cS *CoreSCfg) loadFromJSONCfg(jsnCfg *CoreSJsonCfg) (err error) {
	if jsnCfg == nil {
		return
	}
	if jsnCfg.Caps != nil {
		cS.Caps = *jsnCfg.Caps
	}
	if jsnCfg.Caps_strategy != nil {
		cS.CapsStrategy = *jsnCfg.Caps_strategy
	}
	if jsnCfg.Caps_stats_interval != nil {
		if cS.CapsStatsInterval, err = utils.ParseDurationWithNanosecs(*jsnCfg.Caps_stats_interval); err != nil {
			return
		}
	}
	if jsnCfg.Shutdown_timeout != nil {
		if cS.ShutdownTimeout, err = utils.ParseDurationWithNanosecs(*jsnCfg.Shutdown_timeout); err != nil {
			return
		}
	}
	if jsnCfg.InternalMetricsInterval != nil {
		if cS.InternalMetricsInterval, err = utils.ParseDurationWithNanosecs(*jsnCfg.InternalMetricsInterval); err != nil {
			return
		}
	}
	if jsnCfg.EEsConns != nil {
		cS.EEsConns = make([]string, len(jsnCfg.EEsConns))
		for idx, conn := range jsnCfg.EEsConns {
			// if we have the connection internal we change the name so we can have internal rpc for each subsystem
			cS.EEsConns[idx] = conn
			if conn == utils.MetaInternal {
				cS.EEsConns[idx] = utils.ConcatenatedKey(utils.MetaInternal, utils.MetaEEs)
			}
		}
	}
	if jsnCfg.StatSConns != nil {
		cS.StatSConns = make([]string, len(jsnCfg.StatSConns))
		for idx, conn := range jsnCfg.StatSConns {
			// if we have the connection internal we change the name so we can have internal rpc for each subsystem
			cS.StatSConns[idx] = conn
			if conn == utils.MetaInternal {
				cS.StatSConns[idx] = utils.ConcatenatedKey(utils.MetaInternal, utils.MetaStats)
			}
		}
	}
	return
}

// AsMapInterface returns the config as a map[string]any
func (cS *CoreSCfg) AsMapInterface() map[string]any {
	mp := map[string]any{
		utils.CapsCfg:                    cS.Caps,
		utils.CapsStrategyCfg:            cS.CapsStrategy,
		utils.CapsStatsIntervalCfg:       cS.CapsStatsInterval.String(),
		utils.ShutdownTimeoutCfg:         cS.ShutdownTimeout.String(),
		utils.InternalMetricsIntervalCfg: cS.InternalMetricsInterval.String(),
	}
	if cS.EEsConns != nil {
		eesConns := make([]string, len(cS.EEsConns))
		for i, item := range cS.EEsConns {
			eesConns[i] = item
			if item == utils.ConcatenatedKey(utils.MetaInternal, utils.MetaAttributes) {
				eesConns[i] = utils.MetaInternal
			}
		}
		mp[utils.EEsConnsCfg] = eesConns
	}
	if cS.StatSConns != nil {
		statsConns := make([]string, len(cS.StatSConns))
		for i, item := range cS.StatSConns {
			statsConns[i] = item
			if item == utils.ConcatenatedKey(utils.MetaInternal, utils.MetaAttributes) {
				statsConns[i] = utils.MetaInternal
			}
		}
		mp[utils.StatSConnsCfg] = statsConns
	}
	if cS.CapsStatsInterval == 0 {
		mp[utils.CapsStatsIntervalCfg] = "0"
	}
	if cS.ShutdownTimeout == 0 {
		mp[utils.ShutdownTimeoutCfg] = "0"
	}
	return mp
}

// Clone returns a deep copy of CoreSCfg
func (cS CoreSCfg) Clone() *CoreSCfg {
	return &CoreSCfg{
		Caps:                    cS.Caps,
		CapsStrategy:            cS.CapsStrategy,
		CapsStatsInterval:       cS.CapsStatsInterval,
		ShutdownTimeout:         cS.ShutdownTimeout,
		InternalMetricsInterval: cS.InternalMetricsInterval,
		EEsConns:                slices.Clone(cS.EEsConns),
		StatSConns:              slices.Clone(cS.StatSConns),
	}
}
