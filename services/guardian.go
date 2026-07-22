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

package services

import (
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/servmanager"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/guardian"
)

// NewGuardianService instantiates a new GuardianService.
func NewGuardianService(cfg *config.CGRConfig) *GuardianService {
	return &GuardianService{
		cfg: cfg,
	}
}

// GuardianService implements Service interface.
type GuardianService struct {
	cfg    *config.CGRConfig
	locker *guardian.Locker
}

// Start handles the service start.
func (s *GuardianService) Start(shutdown *utils.SyncedChan, registry *servmanager.Registry) error {
	_, err := registry.WaitForServices(shutdown, utils.StateServiceUP,
		[]string{
			utils.LoggerS,
		},
		s.cfg.GeneralCfg().ConnectTimeout)
	if err != nil {
		return err
	}

	s.locker = engine.NewLocker(s.cfg)
	return nil
}

// Reload handles the config changes.
func (s *GuardianService) Reload(_ *utils.SyncedChan, _ *servmanager.Registry) error {
	return nil
}

// Shutdown stops the service.
func (s *GuardianService) Shutdown(registry *servmanager.Registry) error {
	return nil
}

// ServiceName returns the service name
func (s *GuardianService) ServiceName() string {
	return utils.GuardianS
}

// ShouldRun returns if the service should be running.
func (s *GuardianService) ShouldRun() bool {
	return true
}

// Locker returns the process Guardian locker.
func (s *GuardianService) Locker() *guardian.Locker {
	return s.locker
}
