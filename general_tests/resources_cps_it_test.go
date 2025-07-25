//go:build performance

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
	"bytes"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

func TestLoadResourceProcessEvent(t *testing.T) {
	var dbConfig engine.DBCfg
	switch *utils.DBType {
	case utils.MetaInternal:
		dbConfig = engine.DBCfg{
			DataDB: &engine.DBParams{
				Type: utils.StringPointer(utils.MetaInternal),
			},
			StorDB: &engine.DBParams{
				Type: utils.StringPointer(utils.MetaInternal),
			},
		}
	case utils.MetaMySQL:
	case utils.MetaMongo, utils.MetaPostgres:
		t.SkipNow()
	default:
		t.Fatal("unsupported dbtype value")
	}
	content := `{
	"general": {
		"log_level": 7,
	},
    "data_db": {								 
    	"db_type": "redis",						
    	"db_port": 6379, 						
    	"db_name": "10", 						
    },
    "stor_db": {
    	"db_password": "CGRateS.org",
    },
	"apiers": {
		"enabled": true
	},
	"stats": {
		"enabled": true,
		"store_interval": "-1",
	},
    	"resources": {
    	"enabled": true,
    	"store_interval": "-1",
    },
  	"thresholds": {
  		"enabled": true,
		"store_interval": "-1",
       },
	}`

	ng := engine.TestEngine{
		ConfigJSON: content,
		DBCfg:      dbConfig,
		LogBuffer:  bytes.NewBuffer(nil),
	}
	client, _ := ng.Run(t)
	t.Run("SetResourceProfile", func(t *testing.T) {
		var result string
		for i := 1; i <= 10; i++ {
			rls := &engine.ResourceProfileWithAPIOpts{
				ResourceProfile: &engine.ResourceProfile{
					Tenant:            "cgrates.org",
					ID:                fmt.Sprintf("RES_%d", i),
					FilterIDs:         []string{fmt.Sprintf("*string:~*req.Account:100%d", i)},
					UsageTTL:          -1,
					Limit:             -1,
					AllocationMessage: "Account1Channels",
					Weight:            20,
					ThresholdIDs:      []string{utils.MetaNone},
				},
			}
			if err := client.Call(context.Background(), utils.APIerSv1SetResourceProfile, rls, &result); err != nil {
				t.Error(err)
			} else if result != utils.OK {
				t.Error("Unexpected reply returned", result)
			}
		}
	})

	t.Run("AllocateResources", func(t *testing.T) {
		ticker := time.NewTicker(time.Second / time.Duration(*count))
		defer ticker.Stop()
		jobs := make(chan int, *count)
		for i := 1; i <= *count; i++ {
			jobs <- i
		}
		close(jobs)
		numWrk := 50
		var wg sync.WaitGroup
		latencies := make(chan time.Duration, *count)
		totalCall := time.Now()
		for range numWrk {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					<-ticker.C
					callStart := time.Now()
					args := &utils.CGREvent{
						Tenant: "cgrates.org",
						ID:     utils.UUIDSha1Prefix(),
						Event: map[string]any{
							utils.AccountField: fmt.Sprintf("100%d", ((i-1)%10)+1),
							utils.Destination:  "3420340",
						},
						APIOpts: map[string]any{
							utils.OptsResourcesUnits:   6,
							utils.OptsResourcesUsageID: utils.GenUUID(),
						},
					}
					var reply string
					if err := client.Call(context.Background(), utils.ResourceSv1AllocateResources,
						args, &reply); err != nil {
						t.Errorf("Error processing event %d: %v", i, err)
						continue
					}
					latencies <- time.Since(callStart)
				}

			}()
		}
		wg.Wait()
		totalDuration := time.Since(totalCall)
		close(latencies)
		latencySlice := make([]time.Duration, 0, *count)
		for latency := range latencies {
			latencySlice = append(latencySlice, latency)
		}
		actualThroughput := float64(len(latencySlice)) / totalDuration.Seconds()
		slices.Sort(latencySlice)
		t.Logf("Performance Load Test Results")
		t.Logf("Target Rate: %d events/sec", *count)
		t.Logf("Actual Throughput:  %.2f events/sec", actualThroughput)
		t.Logf("Total Duration:     %v", totalDuration)

		// Calculate percentiles
		p50Index := int(float64(len(latencySlice)) * 0.50)
		p90Index := int(float64(len(latencySlice)) * 0.90)
		p99Index := int(float64(len(latencySlice)) * 0.99)

		t.Logf("p50 Latency: %v", latencySlice[p50Index])
		t.Logf("p90 Latency: %v", latencySlice[p90Index])
		t.Logf("p99 Latency: %v", latencySlice[p99Index])
	})

}
