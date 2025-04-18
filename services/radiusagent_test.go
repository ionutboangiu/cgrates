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
package services

import (
	"reflect"
	"sync"
	"testing"

	"github.com/cgrates/birpc"
	"github.com/cgrates/cgrates/agents"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

// TestLoaderSCoverage for cover testing
func TestRadiusAgentCoverage(t *testing.T) {
	cfg := config.NewDefaultCGRConfig()
	cfg.SessionSCfg().Enabled = true
	cfg.SessionSCfg().ListenBijson = ""
	filterSChan := make(chan *engine.FilterS, 1)
	filterSChan <- nil
	shdChan := utils.NewSyncedChan()
	chS := engine.NewCacheS(cfg, nil, nil)
	cacheSrv, err := engine.NewService(chS)
	if err != nil {
		t.Fatal(err)
	}
	cacheSChan := make(chan birpc.ClientConnector, 1)
	cacheSChan <- cacheSrv
	srvDep := map[string]*sync.WaitGroup{utils.DataDB: new(sync.WaitGroup)}
	srv := NewRadiusAgent(cfg, filterSChan, shdChan, nil, nil, srvDep)
	if srv.IsRunning() {
		t.Errorf("Expected service to be down")
	}
	srv2 := RadiusAgent{
		cfg:         cfg,
		filterSChan: filterSChan,
		shdChan:     shdChan,
		rad:         &agents.RadiusAgent{},
		connMgr:     nil,
		srvDep:      srvDep,
	}
	if !srv2.IsRunning() {
		t.Errorf("Expected service to be running")
	}
	serviceName := srv2.ServiceName()
	if !reflect.DeepEqual(serviceName, utils.RadiusAgent) {
		t.Errorf("\nExpecting <%+v>,\n Received <%+v>", utils.RadiusAgent, serviceName)
	}
	shouldRun := srv2.ShouldRun()
	if !reflect.DeepEqual(shouldRun, false) {
		t.Errorf("\nExpecting <false>,\n Received <%+v>", shouldRun)
	}
	srv2.stopChan = make(chan struct{}, 1)
	srv2.Shutdown()
	if srv.IsRunning() {
		t.Errorf("Expected service to be down")
	}
}
