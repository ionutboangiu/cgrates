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

/*
import (
	"net/rpc"
	"net/rpc/jsonrpc"
	"path"
	"testing"
	"time"

	"github.com/cgrates/cgrates/apier/v2"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

var (
	destCfgPath string
	destCfg *config.CGRConfig
	destRPC *birpc.Client

	sTestDestManag = []func (t *testing.T){
	testDestManagInitCfg,
	testDestManagResetDataDb,
	testDestManagResetStorDb,
	testDestManagStartEngine,
	testDestManagRpcConn,
	testDestManagLoadTariffPlanFromFolderAll,
	testDestManagAllDestinationLoaded,
	testDestManagLoadTariffPlanFromFolderRemoveSome,
	testDestManagRemoveSomeDestinationLoaded,
	testDestManagLoadTariffPlanFromFolderRemoveSomeFlush,
	testDestManagRemoveSomeFlushDestinationLoaded,
	testDestManagLoadTariffPlanFromFolderAddBack,
	testDestManagAddBackDestinationLoaded,
	testDestManagLoadTariffPlanFromFolderAddOne,
	testDestManagAddOneDestinationLoaded,
	testDestManagCacheWithGetCache,
	testDestManagCacheWithGetCost,
	}
)

func TestDestManag(t *testing.T) {
	for _, stest := range sTestDestManag {
		t.Run("TestDestManag", stest)
	}
}
func testDestManagInitCfg(t *testing.T) {
	destCfgPath = path.Join(*utils.DataDir, "conf", "samples", "tutmysql")
	// Init config first
	var err error
	destCfg, err = config.NewCGRConfigFromPath(destCfgPath)
	if err != nil {
		t.Error(err)
	}
	destCfg.DataFolderPath = *utils.DataDir // Share DataFolderPath through config towards StoreDb for Flush()
	config.SetCgrConfig(destCfg)
}

// Remove data in both rating and accounting db
func testDestManagResetDataDb(t *testing.T) {
	if err := engine.InitDataDb(destCfg); err != nil {
		t.Fatal(err)
	}
}

// Wipe out the cdr database
func testDestManagResetStorDb(t *testing.T) {
	if err := engine.InitStorDb(destCfg); err != nil {
		t.Fatal(err)
	}
}

// Start CGR Engine
func testDestManagStartEngine(t *testing.T) {
	if _, err := engine.StopStartEngine(destCfgPath, *utils.WaitRater); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func testDestManagRpcConn(t *testing.T) {
	var err error
	destRPC, err = jsonrpc.Dial(utils.TCP, destCfg.RPCJSONListen) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal(err)
	}
}

// Load the tariff plan, creating accounts and their balances
func testDestManagLoadTariffPlanFromFolderAll(t *testing.T) {
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "alldests")}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups
}


func testDestManagAllDestinationLoaded(t *testing.T) {
	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 6 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 9 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}


func testDestManagLoadTariffPlanFromFolderRemoveSome(t *testing.T) {
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "removesome")}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups
}

func testDestManagRemoveSomeDestinationLoaded(t *testing.T) {
	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 6 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 9 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}

func testDestManagLoadTariffPlanFromFolderRemoveSomeFlush(t *testing.T) {
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "removesome"), FlushDb: true}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups
}

func testDestManagRemoveSomeFlushDestinationLoaded(t *testing.T) {
	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 4 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 5 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}

func testDestManagLoadTariffPlanFromFolderAddBack(t *testing.T) {
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "addback")}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups
}

func testDestManagAddBackDestinationLoaded(t *testing.T) {
	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 6 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 9 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}

func testDestManagLoadTariffPlanFromFolderAddOne(t *testing.T) {
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "addone")}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups
}

func testDestManagAddOneDestinationLoaded(t *testing.T) {
	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 7 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 10 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}

func testDestManagCacheWithGetCache(t *testing.T) {
	if err := engine.InitDataDb(destCfg); err != nil {
		t.Fatal(err)
	}
	var reply string
	if err := destRPC.Call(context.Background(),utils.APIerSv1ReloadCache, utils.AttrReloadCache{}, &reply); err != nil {
		t.Error("Got error on APIerSv1.ReloadCache: ", err.Error())
	} else if reply != utils.OK {
		t.Errorf("Calling APIerSv1.ReloadCache received: %+v", reply)
	}
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "cacheall"), FlushDb: true}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups

	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 1 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var rcvStats utils.CacheStats
	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 2 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}

	attrs = &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "cacheone"), FlushDb: true}
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups

	dests = make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 1 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	if err := destRPC.Call(context.Background(),utils.APIerSv1GetCacheStats, utils.AttrCacheStats{}, &rcvStats); err != nil {
		t.Error("Got error on APIerSv1.GetCacheStats: ", err.Error())
	} else if rcvStats.Destinations != 1 {
		t.Errorf("Calling APIerSv1.GetCacheStats received: %+v", rcvStats)
	}
}

func testDestManagCacheWithGetCost(t *testing.T) {
	if err := engine.InitDataDb(destCfg); err != nil {
		t.Fatal(err)
	}
	var reply string
	if err := destRPC.Call(context.Background(),utils.APIerSv1ReloadCache, utils.AttrReloadCache{}, &reply); err != nil {
		t.Error("Got error on APIerSv1.ReloadCache: ", err.Error())
	} else if reply != utils.OK {
		t.Errorf("Calling APIerSv1.ReloadCache received: %+v", reply)
	}
	attrs := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "cacheall"), FlushDb: true}
	var destLoadInst utils.LoadInstance
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups

	dests := make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 1 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	var cc engine.CallCost
	cd := &engine.CallDescriptor{
		Tenant:      "cgrates.org",
		Category:    "call",
		Account:     "test",
		Destination: "1002",
		TimeStart:   time.Date(2016, 2, 24, 0, 0, 0, 0, time.UTC),
		TimeEnd:     time.Date(2016, 2, 24, 0, 0, 10, 0, time.UTC),
	}
	if err := destRPC.Call(context.Background(),utils.ResponderGetCost, cd, &cc); err != nil {
		t.Error(err)
	} else if cc.Cost != 1.6667 {
		t.Error("Empty loadId received, loadInstance: ", utils.ToIJSON(cc))
	}

	attrs = &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*utils.DataDir, "tariffplans", "test", "destinations", "cacheone"), FlushDb: true}
	if err := destRPC.Call(context.Background(),utils.APIerSv2LoadTariffPlanFromFolder, attrs, &destLoadInst); err != nil {
		t.Error(err)
	}
	time.Sleep(time.Duration(*utils.WaitRater) * time.Millisecond) // Give time for scheduler to execute topups

	dests = make([]*engine.Destination, 0)
	if err := destRPC.Call(context.Background(),utils.APIerSv2GetDestinations, &v2.AttrGetDestinations{DestinationIDs: []string{}}, &dests); err != nil {
		t.Error("Got error on APIerSv2.GetDestinations: ", err.Error())
	} else if len(dests) != 1 {
		t.Errorf("Calling APIerSv2.GetDestinations got reply: %v", utils.ToIJSON(dests))
	}

	if err := destRPC.Call(context.Background(),utils.ResponderGetCost, cd, &cc); err.Error() != utils.ErrUnauthorizedDestination.Error() {
		t.Error(err)
	}
}
*/
