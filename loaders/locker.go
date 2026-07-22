/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>
*/

package loaders

import (
	"os"

	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/guardian"
)

type loaderLocker struct {
	path     string
	loaderID string
	memory   *guardian.Locker
}

func newLoaderLocker(path, loaderID string, memory *guardian.Locker) loaderLocker {
	locker := loaderLocker{path: path}
	if path == utils.MetaMemory {
		locker.loaderID = loaderID
		locker.memory = memory
	}
	return locker
}

func (l loaderLocker) lock() (func(), error) {
	switch l.path {
	case "":
		return func() {}, nil
	case utils.MetaMemory:
		return l.memory.Lock(utils.ConcatenatedKey(utils.LoaderS, l.loaderID)), nil
	}
	file, err := os.OpenFile(l.path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	return func() { _ = os.Remove(l.path) }, nil
}

func (l loaderLocker) forceUnlock() error {
	if l.path == "" || l.path == utils.MetaMemory {
		return nil
	}
	return os.Remove(l.path)
}

func (l loaderLocker) locked() (bool, error) {
	if l.path == "" || l.path == utils.MetaMemory {
		return false, nil
	}
	if _, err := os.Stat(l.path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l loaderLocker) isLockFile(path string) bool {
	return l.path != "" && l.path != utils.MetaMemory && path == l.path
}
