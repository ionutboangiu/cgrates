/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or56
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>
*/

package engine

import (
	"reflect"
	"testing"
	"time"

	"github.com/cgrates/cgrates/utils"
)

func TestModelsAsMapStringInterface(t *testing.T) {
	testCdrSql := CDRsql{
		ID: 1,
		// Cgrid:       "testCgrID1",
		RunID:       "testRunID",
		OriginHost:  "testOriginHost",
		Source:      "testSource",
		OriginID:    "testOriginId",
		TOR:         "testTOR",
		RequestType: "testRequestType",
		Tenant:      "cgrates.org",
		Category:    "testCategory",
		Account:     "testAccount",
		Subject:     "testSubject",
		Destination: "testDestination",
		SetupTime:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		AnswerTime:  utils.TimePointer(time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC)),
		Usage:       2,
		ExtraFields: "extraFields",
		CostSource:  "testCostSource",
		Cost:        2,
		CostDetails: "testCostDetails",
		ExtraInfo:   "testExtraInfo",
		CreatedAt:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		UpdatedAt:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		DeletedAt:   utils.TimePointer(time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC)),
	}
	expected := map[string]any{
		// "cgrid":        testCdrSql.Cgrid,
		"run_id":       testCdrSql.RunID,
		"originHost":   testCdrSql.OriginHost,
		"source":       testCdrSql.Source,
		"origin_id":    testCdrSql.OriginID,
		"tor":          testCdrSql.TOR,
		"request_type": testCdrSql.RequestType,
		"tenant":       testCdrSql.Tenant,
		"category":     testCdrSql.Category,
		"account":      testCdrSql.Account,
		"subject":      testCdrSql.Subject,
		"destination":  testCdrSql.Destination,
		"setup_time":   testCdrSql.SetupTime,
		"answer_time":  testCdrSql.AnswerTime,
		"usage":        testCdrSql.Usage,
		"extraFields":  testCdrSql.ExtraFields,
		"cost_source":  testCdrSql.CostSource,
		"cost":         testCdrSql.Cost,
		"cost_details": testCdrSql.CostDetails,
		"extra_info":   testCdrSql.ExtraInfo,
		"created_at":   testCdrSql.CreatedAt,
		"updated_at":   testCdrSql.UpdatedAt,
	}
	result := testCdrSql.AsMapStringInterface()
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("\nExpected <%+v>, \nReceived <%+v>", expected, result)
	}
}

func TestCDRsqlTableName(t *testing.T) {
	cdrSql := &CDRsql{
		ID:          1,
		RunID:       "testRunID",
		OriginHost:  "testOriginHost",
		Source:      "testSource",
		OriginID:    "testOriginId",
		TOR:         "testTOR",
		RequestType: "testRequestType",
		Tenant:      "cgrates.org",
		Category:    "testCategory",
		Account:     "testAccount",
		Subject:     "testSubject",
		Destination: "testDestination",
		SetupTime:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		AnswerTime:  utils.TimePointer(time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC)),
		Usage:       2,
		ExtraFields: "extraFields",
		CostSource:  "testCostSource",
		Cost:        2,
		CostDetails: "testCostDetails",
		ExtraInfo:   "testExtraInfo",
		CreatedAt:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		UpdatedAt:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		DeletedAt:   utils.TimePointer(time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC)),
	}
	rcv := cdrSql.TableName()
	if !reflect.DeepEqual(rcv, utils.CDRsTBL) {
		t.Errorf("Expected <%v>, Received <%v>", utils.CDRsTBL, rcv)
	}
}
func TestSessionCostsSQLTableName(t *testing.T) {
	sessCostSql := &SessionCostsSQL{
		ID:          1,
		RunID:       "testRunID",
		OriginHost:  "testOriginHost",
		OriginID:    "testOriginId",
		CostSource:  "testCostSource",
		Usage:       2,
		CostDetails: "testCostDetails",
		CreatedAt:   time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC),
		DeletedAt:   utils.TimePointer(time.Date(2021, 3, 3, 3, 3, 3, 3, time.UTC)),
	}
	rcv := sessCostSql.TableName()
	if !reflect.DeepEqual(rcv, utils.SessionCostsTBL) {
		t.Errorf("Expected <%v>, Received <%v>", utils.SessionCostsTBL, rcv)
	}
}
func TestTBLVersionTableName(t *testing.T) {
	tblVer := &TBLVersion{
		ID:      1,
		Item:    "testItem",
		Version: 14,
	}
	rcv := tblVer.TableName()
	if !reflect.DeepEqual(rcv, utils.TBLVersions) {
		t.Errorf("Expected <%v>, Received <%v>", utils.TBLVersions, rcv)
	}
}
