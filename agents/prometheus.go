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

package agents

import (
	"fmt"
	"net/http"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusAgent ...
type PrometheusAgent struct {
	cfg      *config.CGRConfig
	filters  *engine.FilterS
	cm       *engine.ConnManager
	shutdown *utils.SyncedChan
	reg      *prometheus.Registry

	// Metric descriptors
	statsDesc *prometheus.Desc
}

func NewPrometheusAgent(cfg *config.CGRConfig, filters *engine.FilterS, cm *engine.ConnManager,
	shutdown *utils.SyncedChan) *PrometheusAgent {
	reg := prometheus.NewRegistry()
	pa := &PrometheusAgent{
		cfg:      cfg,
		filters:  filters,
		cm:       cm,
		shutdown: shutdown,
		reg:      reg,

		statsDesc: prometheus.NewDesc(
			"cgrates_stats_metrics",
			"Current values for StatQueue metrics",
			[]string{"tenant", "queue", "metric"},
			nil,
		),
	}
	reg.MustRegister(pa)
	if cfg.PrometheusAgentCfg().CollectGoMetrics {
		reg.MustRegister(collectors.NewGoCollector())
	}
	if cfg.PrometheusAgentCfg().CollectProcessMetrics {
		reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	return pa
}

// Describe implements prometheus.Collector.
func (pa *PrometheusAgent) Describe(ch chan<- *prometheus.Desc) {
	ch <- pa.statsDesc
}

// Collect implements prometheus.Collector.
func (pa *PrometheusAgent) Collect(ch chan<- prometheus.Metric) {
	if len(pa.cfg.PrometheusAgentCfg().StatQueueIDs) > 0 {
		for _, sqID := range pa.cfg.PrometheusAgentCfg().StatQueueIDs {
			tenantID := utils.NewTenantID(sqID)
			var metrics map[string]float64
			err := pa.cm.Call(context.Background(), pa.cfg.PrometheusAgentCfg().StatSConns,
				utils.StatSv1GetQueueFloatMetrics,
				&utils.TenantIDWithAPIOpts{
					TenantID: tenantID,
				}, &metrics)
			if err != nil {
				if err.Error() != utils.ErrNotFound.Error() {
					utils.Logger.Err(fmt.Sprintf(
						"<%s> failed to retrieve metrics for StatQueue %q: %v",
						utils.PrometheusAgent, sqID, err))
					pa.shutdown.CloseOnce()
					return
				}
				continue
			}
			for metricID, val := range metrics {
				ch <- prometheus.MustNewConstMetric(
					pa.statsDesc,
					prometheus.GaugeValue,
					val,
					tenantID.Tenant, tenantID.ID, metricID,
				)
			}
		}
	}
}

// ServeHTTP implements http.Handler.
func (pa *PrometheusAgent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	promhttp.HandlerFor(pa.reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
