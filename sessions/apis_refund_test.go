// Copyright ITsysCOM GmbH
// SPDX-License-Identifier: AGPL-3.0-or-later

package sessions

import (
	"encoding/json"
	"testing"

	"github.com/cgrates/birpc"
	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

type refundTestClient func(ctx *context.Context, method string, args, reply any) error

func (client refundTestClient) Call(ctx *context.Context, method string, args, reply any) error {
	return client(ctx, method, args, reply)
}

func testRefundCharges(t *testing.T) (*utils.EventCharges, map[string]any) {
	t.Helper()
	charges := &utils.EventCharges{
		Abstracts: utils.NewDecimal(int64(1), 0),
		Concretes: utils.NewDecimal(1, 0),
		Charges: []*utils.ChargeEntry{
			{ChargingID: "abstract", CompressFactor: 1},
		},
		Accounting: map[string]*utils.AccountCharge{
			"abstract": {
				AccountID:       "1001",
				BalanceID:       utils.MetaMockAbstract,
				Units:           utils.NewDecimal(1, 0),
				JoinedChargeIDs: []string{"concrete"},
			},
			"concrete": {
				AccountID: "1001",
				BalanceID: "MONETARY",
				Units:     utils.NewDecimal(1, 0),
			},
		},
		Accounts: map[string]*utils.Account{
			"1001": {
				ID: "1001",
				Balances: map[string]*utils.Balance{
					"MONETARY": {
						ID:    "MONETARY",
						Units: utils.NewDecimal(9, 0),
					},
				},
			},
		},
	}
	encoded, err := json.Marshal(charges)
	if err != nil {
		t.Fatal(err)
	}
	var chargesMap map[string]any
	if err := json.Unmarshal(encoded, &chargesMap); err != nil {
		t.Fatal(err)
	}
	return charges, chargesMap
}

func TestEventChargesFromInterface(t *testing.T) {
	typed, chargesMap := testRefundCharges(t)
	if got, err := eventChargesFromInterface(typed); err != nil {
		t.Fatal(err)
	} else if got != typed {
		t.Fatal("typed EventCharges pointer was replaced")
	}

	got, err := eventChargesFromInterface(chargesMap)
	if err != nil {
		t.Fatal(err)
	} else if got.Concretes.Compare(utils.NewDecimal(1, 0)) != 0 {
		t.Fatalf("decoded Concretes = %s, want 1", got.Concretes)
	}

	if _, err := eventChargesFromInterface("invalid"); err == nil {
		t.Fatal("expected invalid EventCharges type error")
	}
}

func TestSessionSBiRPCv1ProcessEventRefund(t *testing.T) {
	cfg := config.NewDefaultCGRConfig()
	cfg.CacheCfg().Partitions[utils.CacheRPCResponses].Limit = 0
	data, err := engine.NewInternalDB(nil, nil, nil, cfg.DbCfg().Items)
	if err != nil {
		t.Fatal(err)
	}
	locker := engine.NewLocker(cfg)
	connMgr := engine.NewConnManager(cfg)
	dbCM := engine.NewDBConnManager(map[string]engine.DataDB{utils.MetaDefault: data}, cfg.DbCfg())
	dm := engine.NewDataManager(dbCM, cfg, connMgr, locker)
	cacheS := engine.NewCacheS(cfg, dm, connMgr, nil, locker)
	dm.SetCache(cacheS)
	connMgr.SetCache(cacheS)
	fltrS := engine.NewFilterS(cfg, nil, dm)

	refundCalls := 0
	client := refundTestClient(func(ctx *context.Context, method string, args, reply any) error {
		if method != utils.AccountSv1RefundCharges {
			t.Fatalf("unexpected method %s", method)
		}
		refundCalls++
		charges := args.(*utils.APIEventCharges).EventCharges
		if charges.Concretes.Compare(utils.NewDecimal(1, 0)) != 0 {
			t.Fatalf("refunded Concretes = %s, want 1", charges.Concretes)
		}
		return utils.ErrNotImplemented
	})
	internal := make(chan birpc.ClientConnector, 1)
	internal <- client
	connID := utils.ConcatenatedKey(utils.MetaInternal, utils.MetaAccounts)
	cfg.SessionSCfg().Conns[utils.MetaAccounts] = []*config.DynamicConns{{ConnIDs: []string{connID}}}
	connMgr.AddInternalConn(connID, utils.AccountSv1, internal)
	sessionS := NewSessionS(cfg, dm, cacheS, fltrS, connMgr)
	_, chargesMap := testRefundCharges(t)

	event := &utils.CGREvent{
		Tenant: "cgrates.org",
		Event:  map[string]any{utils.AccountField: "1001"},
		APIOpts: map[string]any{
			utils.MetaOriginID:        "refund",
			utils.MetaRefund:          true,
			utils.MetaAccountsCost:    chargesMap,
			utils.OptsSesBlockerError: true,
		},
	}
	var reply V1ProcessEventReply
	if err := sessionS.BiRPCv1ProcessEvent(context.TODO(), event, &reply); err != utils.ErrNotImplemented {
		t.Fatalf("refund error = %v, want %v", err, utils.ErrNotImplemented)
	}
	if refundCalls != 1 {
		t.Fatalf("refund calls = %d, want 1", refundCalls)
	}

	delete(event.APIOpts, utils.MetaAccountsCost)
	event.ID = ""
	if err := sessionS.BiRPCv1ProcessEvent(context.TODO(), event, &reply); err == nil ||
		err.Error() != utils.NewErrMandatoryIeMissing(utils.MetaAccountsCost).Error() {
		t.Fatalf("missing ledger error = %v", err)
	}
	if refundCalls != 1 {
		t.Fatal("refund called with missing ledger")
	}
}
