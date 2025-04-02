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

package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cgrates/birpc/context"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/guardian"
	"github.com/cgrates/cgrates/utils"
)

type ReplicationManager struct {
	cm  *ConnManager
	rb  *ReplicationBuffer
	cfg *config.DataDbCfg
}

func NewReplicationManager(cm *ConnManager) *ReplicationManager {
	rm := &ReplicationManager{
		cm:  cm,
		cfg: config.CgrConfig().DataDbCfg(),
	}
	rm.rb = NewReplicationBuffer(rm.cfg.RplInterval)
	return rm
}

// Replicate handles all replication logic in one place
func (rm *ReplicationManager) Replicate(objType, objID, method string, args any, item *config.ItemOpt) error {
	if !item.Replicate {
		return nil
	}

	replicateFunc := func() error {
		return rm.replicate(objType, objID, method, args)
	}

	if rm.rb != nil {
		_, methodName, _ := strings.Cut(method, utils.NestingSep)
		key := methodName + utils.Underline + objType + objID
		rm.rb.Enqueue(key, replicateFunc)
		return nil
	}

	return replicateFunc()
}

func (rm *ReplicationManager) replicate(objType, objID, method string, args any) error {
	// the reply is string for Set/Remove APIs
	// ignored in favor of the error
	var reply string
	if !rm.cfg.RplFiltered {
		// is not partial so send to all defined connections
		return utils.CastRPCErr(rm.cm.Call(context.TODO(), rm.cfg.RplConns, method, args, &reply))
	}
	// is partial so get all the replicationHosts from cache based on object Type and ID
	// alp_cgrates.org:ATTR1
	rplcHostIDsIfaces := Cache.tCache.GetGroupItems(utils.CacheReplicationHosts, objType+objID)
	rplcHostIDs := make(utils.StringSet)
	for _, hostID := range rplcHostIDsIfaces {
		rplcHostIDs.Add(hostID.(string))
	}
	// using the replication hosts call the method
	return utils.CastRPCErr(rm.cm.CallWithConnIDs(rm.cfg.RplConns, rplcHostIDs,
		method, args, &reply))
}

func (rm *ReplicationManager) Close() {
	if rm.rb != nil {
		rm.rb.Close()
	}
}

// UpdateReplicationFilters will set the connID in cache
func UpdateReplicationFilters(objType, objID, connID string) {
	if connID == utils.EmptyString {
		return
	}
	Cache.SetWithoutReplicate(utils.CacheReplicationHosts, objType+objID+utils.ConcatenatedKeySep+connID, connID, []string{objType + objID},
		true, utils.NonTransactional)
}

// replicate will call Set/Remove APIs on ReplicatorSv1
func replicate(connMgr *ConnManager, connIDs []string, filtered bool, objType, objID, method string, args any) (err error) {
	// the reply is string for Set/Remove APIs
	// ignored in favor of the error
	var reply string
	if !filtered {
		// is not partial so send to all defined connections
		return utils.CastRPCErr(connMgr.Call(context.TODO(), connIDs, method, args, &reply))
	}
	// is partial so get all the replicationHosts from cache based on object Type and ID
	// alp_cgrates.org:ATTR1
	rplcHostIDsIfaces := Cache.tCache.GetGroupItems(utils.CacheReplicationHosts, objType+objID)
	rplcHostIDs := make(utils.StringSet)
	for _, hostID := range rplcHostIDsIfaces {
		rplcHostIDs.Add(hostID.(string))
	}
	// using the replication hosts call the method
	return utils.CastRPCErr(connMgr.CallWithConnIDs(connIDs, rplcHostIDs,
		method, args, &reply))
}

// replicateMultipleIDs will do the same thing as replicate but uses multiple objectIDs
// used when setting the LoadIDs
// TODO: merge with replicate
func replicateMultipleIDs(connMgr *ConnManager, connIDs []string, filtered bool, objType string, objIDs []string, method string, args any) (err error) {
	// the reply is string for Set/Remove APIs
	// ignored in favor of the error
	var reply string
	if !filtered {
		// is not partial so send to all defined connections
		return utils.CastRPCErr(connMgr.Call(context.TODO(), connIDs, method, args, &reply))
	}
	// is partial so get all the replicationHosts from cache based on object Type and ID
	// combine all hosts in a single set so if we receive a get with one ID in list
	// send all list to that hos
	rplcHostIDs := make(utils.StringSet)
	for _, objID := range objIDs {
		rplcHostIDsIfaces := Cache.tCache.GetGroupItems(utils.CacheReplicationHosts, objType+objID)
		for _, hostID := range rplcHostIDsIfaces {
			rplcHostIDs.Add(hostID.(string))
		}
	}
	// using the replication hosts call the method
	return utils.CastRPCErr(connMgr.CallWithConnIDs(connIDs, rplcHostIDs,
		method, args, &reply))
}

func dumpFailedReplicate(req *ReplicationRequest) error {
	_, methodName, _ := strings.Cut(req.Method, utils.NestingSep)
	key := methodName + utils.Underline + req.ObjType + req.ObjID
	filePath := filepath.Join(req.failedRplDir, key+utils.GOBSuffix)
	return req.WriteToFile(filePath)
}

// NewReplicationRequestFromFile returns ExportEvents from the file
// used only on replay failed post
func NewReplicationRequestFromFile(path string) (*ReplicationRequest, error) {
	var fileContent []byte
	if err := guardian.Guardian.Guard(func() error {
		var err error
		if fileContent, err = os.ReadFile(path); err != nil {
			return err
		}
		return os.Remove(path)
	}, config.CgrConfig().GeneralCfg().LockingTimeout, utils.FileLockPrefix+path); err != nil {
		return nil, err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(fileContent))
	// unmarshall it
	var req *ReplicationRequest
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

// ReplicationRequest used to save the failed post to file
type ReplicationRequest struct {
	ConnIDs      []string
	Filtered     bool
	Path         string
	ObjType      string
	ObjID        string
	Method       string
	Args         any
	connMgr      *ConnManager
	failedRplDir string
}

// WriteToFile writes the events to file.
func (r *ReplicationRequest) WriteToFile(filePath string) (err error) {
	return guardian.Guardian.Guard(func() error {
		f, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer f.Close()
		enc := gob.NewEncoder(f)
		return enc.Encode(r)
	}, config.CgrConfig().GeneralCfg().LockingTimeout, utils.FileLockPrefix+filePath)
}

// ReplayFailedPosts tryies to post cdrs again
func (r *ReplicationRequest) ReplayFailedPosts() error {
	return replicate(connMgr, r.ConnIDs, r.Filtered, r.ObjType, r.ObjID, r.Method, r.Args)
}

type ReplicationBuffer struct {
	mu       sync.Mutex
	ops      map[string]func() error
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewReplicationBuffer(interval time.Duration) *ReplicationBuffer {
	if interval == 0 {
		return nil
	}
	rb := &ReplicationBuffer{
		ops:      make(map[string]func() error),
		interval: interval,
		stop:     make(chan struct{}),
	}
	rb.wg.Add(1)
	go rb.periodicReplicate()
	return rb

}

func (b *ReplicationBuffer) periodicReplicate() {
	defer b.wg.Done()
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.Flush()
		case <-b.stop:
			b.Flush()
			return
		}
	}
}

func (b *ReplicationBuffer) Enqueue(key string, op func() error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ops[key] = op
}

func (b *ReplicationBuffer) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for key, op := range b.ops {
		if err := op(); err != nil {
			// Parse key: methodName_objType+objID
			method, objKey, _ := strings.Cut(key, utils.Underline)
			utils.Logger.Warning(fmt.Sprintf(
				"<DataManager> failed to replicate (method %q, object %q): %v",
				method, objKey, err))
		}

		// Clean up failed replication files if successful.
		path := key + utils.GOBSuffix
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			utils.Logger.Warning(fmt.Sprintf(
				"<DataManager> failed to remove file for %q: %v", key, err))
		}
	}
	b.ops = make(map[string]func() error)
}

func (b *ReplicationBuffer) Close() {
	close(b.stop)
	b.wg.Done()
}
