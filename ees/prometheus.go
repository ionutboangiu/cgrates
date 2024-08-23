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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/cores"
	"github.com/cgrates/cgrates/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusEE struct {
	subsystem string

	mu        sync.Mutex
	registry  *prometheus.Registry
	gauges    map[string]prometheus.Gauge
	counters  map[string]prometheus.Counter
	summaries map[string]prometheus.Summary

	server *http.Server

	cfg  *config.EventExporterCfg
	dc   *utils.SafeMapStorage
	reqs *concReq
}

func newPrometheusEE(cfg *config.EventExporterCfg, dc *utils.SafeMapStorage, muxes ...*http.ServeMux) (*prometheusEE, error) {
	pstr := &prometheusEE{
		cfg:       cfg,
		dc:        dc,
		reqs:      newConcReq(cfg.ConcurrentRequests),
		registry:  prometheus.NewRegistry(),
		gauges:    make(map[string]prometheus.Gauge),
		counters:  make(map[string]prometheus.Counter),
		summaries: make(map[string]prometheus.Summary),
	}
	lastPathElem := cfg.ID
	if cfg.ExportPath != "/var/spool/cgrates/ees" {
		lastPathElem = cfg.ExportPath
	}

	endpoint := fmt.Sprintf("%s/%s", config.CgrConfig().HTTPCfg().PrometheusURL, lastPathElem)
	utils.Logger.Debug("endpoint: " + endpoint)
	// handler := promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promhttp.HandlerFor(pstr.registry, promhttp.HandlerOpts{Registry: pstr.registry}))
	handler := promhttp.HandlerFor(pstr.registry, promhttp.HandlerOpts{Registry: pstr.registry})
	// mux.Handle(endpoint, promhttp.HandlerFor(pstr.registry, promhttp.HandlerOpts{Registry: pstr.registry}))
	for _, mux := range muxes {
		mux.Handle(endpoint, handler)
	}

	return pstr, nil
}

func (p *prometheusEE) Cfg() *config.EventExporterCfg { return p.cfg }

func (p *prometheusEE) Connect() error { return nil }

func (p *prometheusEE) ExportEvent(content any, key string) error {
	utils.Logger.Debug(fmt.Sprintf("content: %T, %v", content, utils.ToJSON(content)))
	metrics, ok := content.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid content type: expected map[string]any")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for mID, mVal := range metrics {
		if strings.HasSuffix(mID, "_total") {
			if err := p.handleCounter(mID, mVal); err != nil {
				return err
			}
		} else if mID == "go_gc_duration_seconds" {
			if err := p.handleSummaries(mID, mVal); err != nil {
				return err
			}
		} else {
			if err := p.handleGauge(mID, mVal); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *prometheusEE) handleGauge(mID string, mVal any) error {
	gauge, ok := p.gauges[mID]
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: mID,
			Help: "Gauge metric",
		})
		p.gauges[mID] = gauge
		if err := p.registry.Register(gauge); err != nil {
			return err
		}
	}

	v, err := utils.IfaceAsFloat64(mVal)
	if err == nil {
		gauge.Set(v)
	} else {
		// For metrics like GoVersion, NodeID, and Version set the value to 1 and use the string as a label (TBA)
		gauge.Set(1)
	}
	return nil
}

func (p *prometheusEE) handleCounter(mID string, mVal any) error {
	counter, ok := p.counters[mID].(prometheus.Counter)
	if !ok {
		counter = prometheus.NewCounter(prometheus.CounterOpts{
			Name: mID,
			Help: "Counter metric",
		})
		p.counters[mID] = counter
		if err := p.registry.Register(counter); err != nil {
			return err
		}
	}

	v, err := utils.IfaceAsFloat64(mVal)
	if err != nil {
		return err
	}
	counter.Add(v)
	return nil
}

func (p *prometheusEE) handleSummaries(mID string, mVal any) error {
	summary, ok := p.summaries[mID].(prometheus.Summary)
	if !ok {
		summary = prometheus.NewSummary(prometheus.SummaryOpts{
			Name: mID,
			Help: "GC duration summary",
		})
		p.summaries[mID] = summary
		if err := p.registry.Register(summary); err != nil {
			return err
		}
	}

	strVal, ok := mVal.(string)
	if !ok {
		return errors.New("invalid summary type")
	}

	var summaryData cores.Summary
	if err := json.Unmarshal([]byte(strVal), &summaryData); err != nil {
		return err
	}

	for i := 0; i < int(summaryData.Count); i++ {
		summary.Observe(summaryData.Sum / float64(summaryData.Count))
	}
	return nil
}

func (p *prometheusEE) Close() error {
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}

type prometheusMetricBatch struct {
	subsystem string
	metrics   []prometheusMetric
}

type prometheusMetric struct {
	id   string
	val  float64
	help string
}

// ignore for now
func (p *prometheusEE) PrepareMap(cgrEv *utils.CGREvent) (any, error) {
	subsys, ok := cgrEv.APIOpts["*subsys"].(string)
	if !ok {
		subsys = ""
	}
	subsys = strings.TrimPrefix(subsys, "*")

	metrics := make([]prometheusMetric, 0, len(cgrEv.Event))
	for key, value := range cgrEv.Event {
		metricID := toSnakeCase(key, '_')
		floatValue, err := utils.IfaceAsFloat64(value)
		if err != nil {
			continue
		}
		metrics = append(metrics, prometheusMetric{
			id:   metricID,
			val:  floatValue,
			help: fmt.Sprintf("Metric %s", metricID),
		})
	}

	return prometheusMetricBatch{
		subsystem: subsys,
		metrics:   metrics,
	}, nil
}

func (p *prometheusEE) PrepareOrderMap(mp *utils.OrderedNavigableMap) (any, error) {
	valMp := make(map[string]any)
	for el := mp.GetFirstElement(); el != nil; el = el.Next() {
		path := el.Value
		nmIt, _ := mp.Field(path)
		path = path[:len(path)-1] // remove the last index
		valMp[strings.Join(path, utils.NestingSep)] = nmIt.String()
	}
	return valMp, nil
}

func (p *prometheusEE) GetMetrics() *utils.SafeMapStorage { return p.dc }

func toSnakeCase(s string, delimiter uint8) string {
	ignore := ""
	screaming := false
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsLow && screaming {
			v += 'A'
			v -= 'a'
		} else if vIsCap && !screaming {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				prevIgnore := ignore != "" && i > 0 && strings.ContainsAny(string(s[i-1]), ignore)
				if !prevIgnore {
					if vIsCap && nextIsLow {
						if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
							n.WriteByte(delimiter)
						}
					}
					n.WriteByte(v)
					if vIsLow || vIsNum || nextIsNum {
						n.WriteByte(delimiter)
					}
					continue
				}
			}
		}

		if (v == ' ' || v == '_' || v == '-' || v == '.') && !strings.ContainsAny(string(v), ignore) {
			// replace space/underscore/hyphen/dot with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}
	return n.String()
}
