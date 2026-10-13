// Copyright (c) 2023 Myntra Designs Private Limited.
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

package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigEnablesNodeRebalanceByDefault(t *testing.T) {
	assert.Equal(t, NodeRebalance{Enabled: true, IntervalSeconds: 60, SettleDelaySeconds: 5}, NewConfig().NodeRebalance)
}

func TestWithNodeRebalanceConfig(t *testing.T) {
	rebalance := NodeRebalance{Enabled: false, IntervalSeconds: 120, SettleDelaySeconds: 10}

	assert.Equal(t, rebalance, NewConfig(WithNodeRebalanceConfig(rebalance)).NodeRebalance)
}

func TestLoadConfigReadsNodeRebalance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf.json")
	contents := `{"NodeRebalance": {"Enabled": true, "IntervalSeconds": 90, "SettleDelaySeconds": 7}}`
	assert.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	assert.Equal(t, NodeRebalance{Enabled: true, IntervalSeconds: 90, SettleDelaySeconds: 7}, LoadConfig(path).NodeRebalance)
}

// Files without the section load as disabled, so consumers that load their own
// configuration must add it to opt in.
func TestLoadConfigWithoutNodeRebalanceLeavesItDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"HttpPort": "8080"}`), 0o600))

	assert.Equal(t, NodeRebalance{}, LoadConfig(path).NodeRebalance)
}

func TestSampleConfigsEnableNodeRebalance(t *testing.T) {
	for _, file := range []string{"conf.json", "conf.docker.json"} {
		t.Run(file, func(t *testing.T) {
			assert.Equal(t, NodeRebalance{Enabled: true, IntervalSeconds: 60, SettleDelaySeconds: 5}, LoadConfig(file).NodeRebalance)
		})
	}
}
