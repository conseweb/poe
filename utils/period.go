/*
Copyright Mojing Inc. 2016 All Rights Reserved.
Written by mint.zhao.chiu@gmail.com. github.com: https://www.github.com/mintzhao

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package utils

import (
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var (
	periodLimits []*PeriodLimit
	once         = &sync.Once{}
	mutex        = &sync.RWMutex{}
)

type PeriodLimit struct {
	Period time.Duration
	Limit  int64
}

// GetPeriodLimits
func GetPeriodLimits() []*PeriodLimit {
	once.Do(func() {
		periodLimits = loadPeriodLimitsFromConfig()
		if len(periodLimits) == 0 {
			panic("periodLimits can't be 0")
		}
	})

	mutex.RLock()
	defer mutex.RUnlock()

	return periodLimits
}

// LoadPeriodLimitsFromConfig
func loadPeriodLimitsFromConfig() []*PeriodLimit {
	mutex.Lock()
	defer mutex.Unlock()

	periodLimitStrs := viper.GetStringMapString("api.period")

	limits := make([]*PeriodLimit, 0)
	for period, limit := range periodLimitStrs {
		periodDuration, err := time.ParseDuration(period)
		if err != nil {
			continue
		}

		limitCnt, err := strconv.ParseInt(limit, 10, 64)
		limits = append(limits, &PeriodLimit{
			Period: periodDuration,
			Limit:  limitCnt,
		})
	}

	return limits
}
