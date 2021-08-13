// Copyright (c) 2021 GPBR Participacoes LTDA.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package config

import (
	"github.com/spf13/viper"
)

const (
	logLevelKey = "log_level"
	devModeKey  = "dev_mode"
)

func init() {
	viper.SetDefault(logLevelKey, "info")
	viper.SetDefault(devModeKey, "false")
	viper.AutomaticEnv()
}

// OperatorCfg represents all possible configurations for the Operator
type OperatorCfg struct {
	// LogLevel represents log verbosity. Overridden to "debug" if DevMode is true.
	LogLevel string
	// DevMode when set to "true" logs in unstructured text instead of JSON.
	DevMode bool
}

// Parse environment variables into a config struct
func Parse() OperatorCfg {
	devMode := viper.GetBool(devModeKey)
	logLvl := viper.GetString(logLevelKey)
	if devMode {
		logLvl = "debug"
	}
	return OperatorCfg{
		LogLevel: logLvl,
		DevMode:  devMode,
	}
}
