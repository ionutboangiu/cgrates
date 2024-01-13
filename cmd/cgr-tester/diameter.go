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

package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/agents"
	"github.com/cgrates/cgrates/utils"
	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
)

func callDiameter(ctx *context.Context, authDur, initDur, updateDur, terminateDur, cdrDur *[]time.Duration,
	reqAuth, reqInit, reqUpdate, reqTerminate, reqCdr *uint64,
	digitMin, digitMax int64, totalUsage time.Duration) (err error) {

	if *digits <= 0 {
		return fmt.Errorf(`"digits" should be bigger than 0`)
	}
	var appendMu sync.Mutex
	acc := strconv.Itoa(int(utils.RandomInteger(digitMin, digitMax)))
	dest := strconv.Itoa(int(utils.RandomInteger(digitMin, digitMax)))

	// ==============================================================================================

	m := diam.NewRequest(diam.CreditControl, 4, nil)
	sessionID := utils.GenUUID()
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sessionID))
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("192.168.1.1"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	m.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Enumerated(1))
	m.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP(avp.DestinationHost, avp.Mbit, 0, datatype.DiameterIdentity("CGR-DA"))
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String("voice@DiamItCCRInit"))
	m.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Date(2018, 10, 4, 14, 42, 20, 0, time.UTC)))
	m.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(0)),   // Subscription-Id-Type
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String(acc)), // Subscription-Id-Data
		}})
	m.NewAVP(avp.ServiceIdentifier, avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(avp.UsedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(avp.ServiceInformation, avp.Mbit, 10415, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(20300, avp.Mbit, 2011, &diam.GroupedAVP{ // IN-Information
				AVP: []*diam.AVP{
					diam.NewAVP(avp.CallingPartyAddress, avp.Mbit, 10415, datatype.UTF8String(acc)), // Calling-Party-Address
					diam.NewAVP(avp.CalledPartyAddress, avp.Mbit, 10415, datatype.UTF8String(dest)), // Called-Party-Address
					diam.NewAVP(20327, avp.Mbit, 2011, datatype.UTF8String(dest)),                   // Real-Called-Number
					diam.NewAVP(20339, avp.Mbit, 2011, datatype.Unsigned32(0)),                      // Charge-Flow-Type
					diam.NewAVP(20302, avp.Mbit, 2011, datatype.UTF8String("")),                     // Calling-Vlr-Number
					diam.NewAVP(20303, avp.Mbit, 2011, datatype.UTF8String("")),                     // Calling-CellID-Or-SAI
					diam.NewAVP(20313, avp.Mbit, 2011, datatype.OctetString("")),                    // Bearer-Capability
					diam.NewAVP(20321, avp.Mbit, 2011, datatype.UTF8String(sessionID)),              // Call-Reference-Number
					diam.NewAVP(20322, avp.Mbit, 2011, datatype.UTF8String("")),                     // MSC-Address
					diam.NewAVP(20324, avp.Mbit, 2011, datatype.Unsigned32(0)),                      // Time-Zone
					diam.NewAVP(20385, avp.Mbit, 2011, datatype.UTF8String("")),                     // Called-Party-NP
					diam.NewAVP(20386, avp.Mbit, 2011, datatype.UTF8String("")),                     // SSP-Time
				},
			}),
		}})

	diamClnt, err := agents.NewDiameterClient(tstCfg.DiameterAgentCfg().Listen, "INTEGRATION_TESTS",
		tstCfg.DiameterAgentCfg().OriginRealm, tstCfg.DiameterAgentCfg().VendorID,
		tstCfg.DiameterAgentCfg().ProductName, utils.DiameterFirmwareRevision,
		tstCfg.DiameterAgentCfg().DictionariesPath, tstCfg.DiameterAgentCfg().ListenNet)
	if err != nil {
		return err
	}

	//
	// SessionSv1InitiateSession
	//
	atomic.AddUint64(reqInit, 1)
	initStartTime := time.Now()
	if err := diamClnt.SendMessage(m); err != nil {
		return err
	}
	msg := diamClnt.ReceivedMessage(tstCfg.GeneralCfg().ReplyTimeout)
	if msg == nil {
		return errors.New("timed out before retrieving message")
	}
	appendMu.Lock()
	*initDur = append(*initDur, time.Since(initStartTime))
	appendMu.Unlock()
	if *verbose {
		log.Printf("Account: <%v>, Destination: <%v>, SessionSv1InitiateSession reply: <%v>", acc, dest, utils.ToJSON(msg))
	}

	//
	// SessionSv1UpdateSession
	//

	m = diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sessionID))
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("192.168.1.1"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	m.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Enumerated(2))
	m.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(1))
	m.NewAVP(avp.DestinationHost, avp.Mbit, 0, datatype.DiameterIdentity("CGR-DA"))
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String("voice@DiamItCCRInit"))
	m.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Date(2018, 10, 4, 14, 42, 20, 0, time.UTC)))
	m.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),   // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String(acc)), // Subscription-Id-Data
		}})
	m.NewAVP(avp.ServiceIdentifier, avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(avp.UsedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(873, avp.Mbit, 10415, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(20300, avp.Mbit, 2011, &diam.GroupedAVP{ // IN-Information
				AVP: []*diam.AVP{
					diam.NewAVP(831, avp.Mbit, 10415, datatype.UTF8String(acc)),        // Calling-Party-Address
					diam.NewAVP(832, avp.Mbit, 10415, datatype.UTF8String(dest)),       // Called-Party-Address
					diam.NewAVP(20327, avp.Mbit, 2011, datatype.UTF8String(dest)),      // Real-Called-Number
					diam.NewAVP(20339, avp.Mbit, 2011, datatype.Unsigned32(0)),         // Charge-Flow-Type
					diam.NewAVP(20302, avp.Mbit, 2011, datatype.UTF8String("")),        // Calling-Vlr-Number
					diam.NewAVP(20303, avp.Mbit, 2011, datatype.UTF8String("")),        // Calling-CellID-Or-SAI
					diam.NewAVP(20313, avp.Mbit, 2011, datatype.OctetString("")),       // Bearer-Capability
					diam.NewAVP(20321, avp.Mbit, 2011, datatype.UTF8String(sessionID)), // Call-Reference-Number
					diam.NewAVP(20322, avp.Mbit, 2011, datatype.UTF8String("")),        // MSC-Address
					diam.NewAVP(20324, avp.Mbit, 2011, datatype.Unsigned32(0)),         // Time-Zone
					diam.NewAVP(20385, avp.Mbit, 2011, datatype.UTF8String("")),        // Called-Party-NP
					diam.NewAVP(20386, avp.Mbit, 2011, datatype.UTF8String("")),        // SSP-Time
				},
			}),
		}})

	var currentUsage time.Duration
	for currentUsage = time.Duration(1 * time.Second); currentUsage < totalUsage; currentUsage += *updateInterval {

		time.Sleep(*updateInterval)
		var temp []*diam.AVP
		temp, err = m.FindAVPsWithPath([]any{"Requested-Service-Unit", "CC-Time"}, dict.UndefinedVendorID)
		if err != nil {
			return err
		}
		if len(temp) != 1 {
			return fmt.Errorf("unexpected number of AVPs returned: %d", len(temp))
		}
		temp[0].Data = datatype.Unsigned32(currentUsage.Seconds())

		atomic.AddUint64(reqUpdate, 1)
		updateStartTime := time.Now()
		if err := diamClnt.SendMessage(m); err != nil {
			return err
		}
		msg = diamClnt.ReceivedMessage(tstCfg.GeneralCfg().ReplyTimeout)
		if msg == nil {
			return errors.New("timed out before retrieving message")
		}
		appendMu.Lock()
		*updateDur = append(*updateDur, time.Since(updateStartTime))
		appendMu.Unlock()
		if *verbose {
			log.Printf("Account: <%v>, Destination: <%v>, SessionSv1UpdateSession reply: <%v>", acc, dest, utils.ToJSON(msg))
		}
		temp, err = m.FindAVPsWithPath([]any{"Used-Service-Unit", "CC-Time"}, dict.UndefinedVendorID)
		if err != nil {
			return err
		}
		if len(temp) != 1 {
			return fmt.Errorf("unexpected number of AVPs returned: %d", len(temp))
		}
		temp[0].Data = datatype.Unsigned32(currentUsage.Seconds())
	}

	// Delay between last update and termination for a more realistic case
	time.Sleep(totalUsage - currentUsage)

	//
	// SessionSv1TerminateSession
	//

	m = diam.NewRequest(diam.CreditControl, 4, nil)
	m.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String(sessionID))
	m.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.DiameterIdentity("192.168.1.1"))
	m.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, datatype.Unsigned32(4))
	m.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Enumerated(3))
	m.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(2))
	m.NewAVP(avp.DestinationHost, avp.Mbit, 0, datatype.DiameterIdentity("CGR-DA"))
	m.NewAVP(avp.DestinationRealm, avp.Mbit, 0, datatype.DiameterIdentity("cgrates.org"))
	m.NewAVP(avp.ServiceContextID, avp.Mbit, 0, datatype.UTF8String("voice@DiamItCCRInit"))
	m.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(time.Date(2018, 10, 4, 14, 42, 20, 0, time.UTC)))
	m.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(450, avp.Mbit, 0, datatype.Enumerated(0)),   // Subscription-Id-Type
			diam.NewAVP(444, avp.Mbit, 0, datatype.UTF8String(acc)), // Subscription-Id-Data
		}})
	m.NewAVP(avp.ServiceIdentifier, avp.Mbit, 0, datatype.Unsigned32(0))
	m.NewAVP(avp.RequestedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(totalUsage.Seconds()))}})
	m.NewAVP(avp.UsedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(420, avp.Mbit, 0, datatype.Unsigned32(0))}})
	m.NewAVP(873, avp.Mbit, 10415, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(20300, avp.Mbit, 2011, &diam.GroupedAVP{ // IN-Information
				AVP: []*diam.AVP{
					diam.NewAVP(831, avp.Mbit, 10415, datatype.UTF8String(acc)),        // Calling-Party-Address
					diam.NewAVP(832, avp.Mbit, 10415, datatype.UTF8String(dest)),       // Called-Party-Address
					diam.NewAVP(20327, avp.Mbit, 2011, datatype.UTF8String(dest)),      // Real-Called-Number
					diam.NewAVP(20339, avp.Mbit, 2011, datatype.Unsigned32(0)),         // Charge-Flow-Type
					diam.NewAVP(20302, avp.Mbit, 2011, datatype.UTF8String("")),        // Calling-Vlr-Number
					diam.NewAVP(20303, avp.Mbit, 2011, datatype.UTF8String("")),        // Calling-CellID-Or-SAI
					diam.NewAVP(20313, avp.Mbit, 2011, datatype.OctetString("")),       // Bearer-Capability
					diam.NewAVP(20321, avp.Mbit, 2011, datatype.UTF8String(sessionID)), // Call-Reference-Number
					diam.NewAVP(20322, avp.Mbit, 2011, datatype.UTF8String("")),        // MSC-Address
					diam.NewAVP(20324, avp.Mbit, 2011, datatype.Unsigned32(0)),         // Time-Zone
					diam.NewAVP(20385, avp.Mbit, 2011, datatype.UTF8String("")),        // Called-Party-NP
					diam.NewAVP(20386, avp.Mbit, 2011, datatype.UTF8String("")),        // SSP-Time
				},
			}),
		}})

	atomic.AddUint64(reqTerminate, 1)
	terminateStartTime := time.Now()
	if err := diamClnt.SendMessage(m); err != nil {
		return err
	}
	msg = diamClnt.ReceivedMessage(tstCfg.GeneralCfg().ReplyTimeout)
	if msg == nil {
		return errors.New("timed out before retrieving message")
	}
	appendMu.Lock()
	*terminateDur = append(*terminateDur, time.Since(terminateStartTime))
	appendMu.Unlock()
	if *verbose {
		log.Printf("Account: <%v>, Destination: <%v>, SessionSv1TerminateSession reply: <%v>", acc, dest, utils.ToJSON(msg))
	}
	return
}
