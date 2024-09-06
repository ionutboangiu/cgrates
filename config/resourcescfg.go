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

package config

import (
	"encoding/json"
	"time"

	"github.com/cgrates/cgrates/utils"
)

type ResourceSOpts struct {
	UsageID  string
	UsageTTL *time.Duration
	Units    float64
}

func (ro *ResourceSOpts) UnmarshalJSON(data []byte) error {
	var temp struct {
		UsageID  *string  `json:"*usageID"`
		UsageTTL *string  `json:"*usageTTL"`
		Units    *float64 `json:"*units"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	if temp.UsageID != nil {
		ro.UsageID = *temp.UsageID
	}
	if temp.UsageTTL != nil {
		ttl, err := utils.ParseDurationWithNanosecs(*temp.UsageTTL)
		if err != nil {
			return err
		}
		ro.UsageTTL = &ttl
	}
	if temp.Units != nil {
		ro.Units = *temp.Units
	}
	return nil
}

// ResourceSCfg stores the configuration settings parsed from the "resources" section.
type ResourceSCfg struct {
	Enabled             bool
	IndexedSelects      bool
	ThresholdSConns     []string
	StoreInterval       time.Duration
	StringIndexedFields *[]string
	PrefixIndexedFields *[]string
	SuffixIndexedFields *[]string
	NestedFields        bool
	Opts                ResourceSOpts
}

func (rc *ResourceSCfg) UnmarshalJSON(data []byte) error {
	var temp struct {
		Enabled             *bool          `json:"enabled"`
		IndexedSelects      *bool          `json:"indexed_selects"`
		ThresholdSConns     []string       `json:"thresholds_conns"`
		StoreInterval       *string        `json:"store_interval"`
		StringIndexedFields []string       `json:"string_indexed_fields"`
		PrefixIndexedFields []string       `json:"prefix_indexed_fields"`
		SuffixIndexedFields []string       `json:"suffix_indexed_fields"`
		NestedFields        *bool          `json:"nested_fields"`
		Opts                *ResourceSOpts `json:"opts"`
	}

	// Ensure opts is updated instead of overwritten.
	temp.Opts = &rc.Opts

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	if temp.Enabled != nil {
		rc.Enabled = *temp.Enabled
	}
	if temp.IndexedSelects != nil {
		rc.IndexedSelects = *temp.IndexedSelects
	}
	if temp.ThresholdSConns != nil {
		rc.ThresholdSConns = subInternalConns(temp.ThresholdSConns, utils.MetaThresholds)
	}
	if temp.StoreInterval != nil {
		var err error
		if rc.StoreInterval, err = utils.ParseDurationWithNanosecs(*temp.StoreInterval); err != nil {
			return err
		}
	}
	if temp.StringIndexedFields != nil {
		rc.StringIndexedFields = &temp.StringIndexedFields
	}
	if temp.PrefixIndexedFields != nil {
		rc.PrefixIndexedFields = &temp.PrefixIndexedFields
	}
	if temp.SuffixIndexedFields != nil {
		rc.SuffixIndexedFields = &temp.SuffixIndexedFields
	}
	if temp.NestedFields != nil {
		rc.NestedFields = *temp.NestedFields
	}
	return nil
}

// subInternalConns substitutes any internal connection ID (*internal) with a modified version
// that includes the subsystem name, using the format *internal:subsystem.
func subInternalConns(conns []string, subsys string) []string {
	updated := make([]string, len(conns))
	for i, c := range conns {
		if c == utils.MetaInternal {
			updated[i] = utils.MetaInternal + utils.ConcatenatedKeySep + subsys
		} else {
			updated[i] = c
		}
	}
	return updated
}

// AsMapInterface returns the config as a map[string]any
func (rlcfg *ResourceSCfg) AsMapInterface() (initialMP map[string]any) {
	opts := map[string]any{
		utils.MetaUsageIDCfg: rlcfg.Opts.UsageID,
		utils.MetaUnitsCfg:   rlcfg.Opts.Units,
	}
	if rlcfg.Opts.UsageTTL != nil {
		opts[utils.MetaUsageTTLCfg] = *rlcfg.Opts.UsageTTL
	}
	initialMP = map[string]any{
		utils.EnabledCfg:        rlcfg.Enabled,
		utils.IndexedSelectsCfg: rlcfg.IndexedSelects,
		utils.NestedFieldsCfg:   rlcfg.NestedFields,
		utils.StoreIntervalCfg:  utils.EmptyString,
		utils.OptsCfg:           opts,
	}
	if rlcfg.ThresholdSConns != nil {
		thresholdSConns := make([]string, len(rlcfg.ThresholdSConns))
		for i, item := range rlcfg.ThresholdSConns {
			thresholdSConns[i] = item
			if item == utils.ConcatenatedKey(utils.MetaInternal, utils.MetaThresholds) {
				thresholdSConns[i] = utils.MetaInternal
			}
		}
		initialMP[utils.ThresholdSConnsCfg] = thresholdSConns
	}
	if rlcfg.StringIndexedFields != nil {
		stringIndexedFields := make([]string, len(*rlcfg.StringIndexedFields))
		copy(stringIndexedFields, *rlcfg.StringIndexedFields)
		initialMP[utils.StringIndexedFieldsCfg] = stringIndexedFields
	}
	if rlcfg.PrefixIndexedFields != nil {
		prefixIndexedFields := make([]string, len(*rlcfg.PrefixIndexedFields))
		copy(prefixIndexedFields, *rlcfg.PrefixIndexedFields)
		initialMP[utils.PrefixIndexedFieldsCfg] = prefixIndexedFields
	}
	if rlcfg.SuffixIndexedFields != nil {
		suffixIndexedFields := make([]string, len(*rlcfg.SuffixIndexedFields))
		copy(suffixIndexedFields, *rlcfg.SuffixIndexedFields)
		initialMP[utils.SuffixIndexedFieldsCfg] = suffixIndexedFields
	}
	if rlcfg.StoreInterval != 0 {
		initialMP[utils.StoreIntervalCfg] = rlcfg.StoreInterval.String()
	}
	return
}

func (resOpts ResourceSOpts) Clone() (cln ResourceSOpts) {
	cln = ResourceSOpts{
		UsageID: resOpts.UsageID,
		Units:   resOpts.Units,
	}
	if resOpts.UsageTTL != nil {
		cln.UsageTTL = new(time.Duration)
		*cln.UsageTTL = *resOpts.UsageTTL
	}
	return
}

// Clone returns a deep copy of ResourceSConfig
func (rlcfg ResourceSCfg) Clone() (cln *ResourceSCfg) {
	cln = &ResourceSCfg{
		Enabled:        rlcfg.Enabled,
		IndexedSelects: rlcfg.IndexedSelects,
		StoreInterval:  rlcfg.StoreInterval,
		NestedFields:   rlcfg.NestedFields,
		Opts:           rlcfg.Opts.Clone(),
	}
	if rlcfg.ThresholdSConns != nil {
		cln.ThresholdSConns = make([]string, len(rlcfg.ThresholdSConns))
		copy(cln.ThresholdSConns, rlcfg.ThresholdSConns)
	}

	if rlcfg.StringIndexedFields != nil {
		idx := make([]string, len(*rlcfg.StringIndexedFields))
		copy(idx, *rlcfg.StringIndexedFields)
		cln.StringIndexedFields = &idx
	}
	if rlcfg.PrefixIndexedFields != nil {
		idx := make([]string, len(*rlcfg.PrefixIndexedFields))
		copy(idx, *rlcfg.PrefixIndexedFields)
		cln.PrefixIndexedFields = &idx
	}
	if rlcfg.SuffixIndexedFields != nil {
		idx := make([]string, len(*rlcfg.SuffixIndexedFields))
		copy(idx, *rlcfg.SuffixIndexedFields)
		cln.SuffixIndexedFields = &idx
	}
	return
}
