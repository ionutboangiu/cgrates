//go:build integration
// +build integration

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

package general_tests

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/utils"
)

func TestPromExporter(t *testing.T) {
	switch *utils.DBType {
	case utils.MetaInternal:
	case utils.MetaMySQL, utils.MetaMongo, utils.MetaPostgres:
		t.SkipNow()
	default:
		t.Fatal("unsupported dbtype value")
	}

	content := `{
"general": {
	"log_level": 7,
},
"data_db": {
	"db_type": "*internal"
},
"stor_db": {
	"db_type": "*internal"
},
"listen": {
	"http": ":2080",
	"http_tls": ":2280"
},
"cores": {
	"internal_metrics_interval": "5s",
	"ees_conns": ["*localhost"]
},
"apiers": {
	"enabled": true
},
"stats": {
	"enabled": true,
	"store_interval": "-1",
	"ees_conns": ["*localhost"]
},
"ees": {
	"enabled": true,
	"cache": {
		"*prometheus": {"limit": -1, "ttl": "0", "precache": true},
	},
	"exporters": [
		{
			"id": "prom_cores",
			"type": "*prometheus",
			"filters": ["*string:~*opts.*subsys:*core"],
			"export_path": "prom1",
			"fields": [
				{"tag": "PrometheusDefaultMetrics","type": "*template", "value": "*prometheusAppMetrics"}
			]
		},
		{
			"id": "prom_stats",
			"type": "*prometheus",
			"filters": ["*string:~*opts.*subsys:*stats"],
			"export_path": "prom2",
			"fields": [
				{"tag": "StatID", "path": "*exp.cgrates_stats_id_label", "type": "*variable", "value": "~*req.StatID"},
				{"tag": "StatTCD", "path": "*exp.cgrates_stats_tcd_seconds", "type": "*variable", "value": "~*req.*tcd{*duration_seconds}"},
				{"tag": "StatTCC", "path": "*exp.cgrates_stats_tcc_units", "type": "*variable", "value": "~*req.*tcc"},
				{"tag": "StatACD", "path": "*exp.cgrates_stats_acd_seconds", "type": "*variable", "value": "~*req.*acd{*duration_seconds}"},
				{"tag": "StatACC", "path": "*exp.cgrates_stats_acc_units", "type": "*variable", "value": "~*req.*acc"},
				{"tag": "StatPDD", "path": "*exp.cgrates_stats_pdd_seconds", "type": "*variable", "value": "~*req.*pdd{*duration_seconds}"},
				{"tag": "StatASR", "path": "*exp.cgrates_stats_asr_units", "type": "*variable", "value": "~*req.*asr"},
				{"tag": "NoSQReqStats", "path": "*exp.cgrates_stats_requests_alt", "type": "*variable", "value": "~*req.*sum#1"},
				{"tag": "NoSQReq", "path": "*exp.cgrates_stats_requests_total", "type": "*constant", "value": "1"}
			]
		}
	]
}
}`

	tpFiles := map[string]string{
		utils.StatsCsv: `
#Tenant[0],Id[1],FilterIDs[2],ActivationInterval[3],QueueLength[4],TTL[5],MinItems[6],Metrics[7],MetricFilterIDs[8],Stored[9],Blocker[10],Weight[11],ThresholdIDs[12]
cgrates.org,testSQ,,,100,-1,0,*tcc;*tcd;*pdd;*asr;*acc;*acd;*sum#1,,,,,*none
`,
	}

	testEnv := TestEnvironment{
		ConfigJSON: content,
		TpFiles:    tpFiles,
		// LogBuffer:  &bytes.Buffer{},
	}
	// defer fmt.Println(testEnv.LogBuffer)
	client, cfg := testEnv.Setup(t, 0)

	t.Run("ProcessStats", func(t *testing.T) {
		for i := range 50 {
			var reply []string
			if err := client.Call(context.Background(), utils.StatSv1ProcessEvent, &utils.CGREvent{
				Tenant: "cgrates.org",
				ID:     fmt.Sprintf("event%d", i),
				Event: map[string]any{
					utils.AnswerTime: time.Date(2024, 8, 22, 14, 25, 0, 0, time.UTC),
					utils.Usage:      time.Duration(rand.Intn(3600)+60) * time.Second,
					utils.Cost:       rand.Float64()*20 + 0.1,
					utils.PDD:        time.Duration(rand.Intn(20)+1) * time.Second,
				}}, &reply); err != nil {
				t.Error(err)
			}
		}
	})

	t.Run("Curl", func(t *testing.T) {
		time.Sleep(6 * time.Second) // wait for the first cores metric export
		for _, exporter := range cfg.EEsCfg().Exporters {
			scrapePromURL(t, exporter.ExportPath)
		}
	})
}

func scrapePromURL(t *testing.T, lastPath string) {
	t.Helper()
	url := fmt.Sprintf("http://localhost:2080/prometheus/%s", lastPath)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	bodyString := string(body)
	fmt.Println(bodyString)
}
