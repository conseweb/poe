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
	"github.com/spf13/viper"
	"gopkg.in/check.v1"
)

type PeriodLimitTest struct {
}

var _ = check.Suite(&PeriodLimitTest{})

func (t *PeriodLimitTest) SetUpSuite(c *check.C) {
	viper.Set("api.period", map[string]string{
		"1m":  "1000",
		"10m": "10000",
	})
}

func (t *PeriodLimitTest) TestGetPeriodLimits(c *check.C) {
	pls := GetPeriodLimits()

	c.Check(len(pls), check.Equals, 2)
}
