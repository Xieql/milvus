// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datanode

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactionExecutor(t *testing.T) {
	t.Run("Test execute", func(t *testing.T) {
		ex := newCompactionExecutor()
		go ex.start(context.TODO())
		ex.execute(newMockCompactor(true))
	})

	t.Run("Test stopTask", func(t *testing.T) {
		ex := newCompactionExecutor()
		mc := newMockCompactor(true)
		ex.executing.Store(UniqueID(1), mc)
		ex.stopTask(UniqueID(1))
	})

	t.Run("Test start", func(t *testing.T) {
		ex := newCompactionExecutor()
		ctx, cancel := context.WithCancel(context.TODO())
		cancel()
		go ex.start(ctx)
	})

	t.Run("Test excuteTask", func(t *testing.T) {
		tests := []struct {
			isvalid bool

			description string
		}{
			{true, "compact return nil"},
			{false, "compact return error"},
		}

		ex := newCompactionExecutor()
		for _, test := range tests {
			t.Run(test.description, func(t *testing.T) {
				if test.isvalid {
					ex.executeTask(newMockCompactor(true))
				} else {
					ex.executeTask(newMockCompactor(false))
				}
			})
		}
	})

	t.Run("Test channel valid check", func(t *testing.T) {
		tests := []struct {
			expected bool
			channel  string
			desc     string
		}{
			{expected: true, channel: "ch1", desc: "no in dropped"},
			{expected: false, channel: "ch2", desc: "in dropped"},
		}
		ex := newCompactionExecutor()
		ex.stopExecutingtaskByVChannelName("ch2")
		for _, test := range tests {
			t.Run(test.desc, func(t *testing.T) {
				assert.Equal(t, test.expected, ex.channelValidateForCompaction(test.channel))
			})
		}
	})

	t.Run("test stop vchannel tasks", func(t *testing.T) {
		ex := newCompactionExecutor()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go ex.start(ctx)
		mc := newMockCompactor(true)
		mc.alwaysWorking = true

		ex.execute(mc)

		// wait for task enqueued
		found := false
		for !found {
			ex.executing.Range(func(key, value interface{}) bool {
				found = true
				return true
			})
		}

		ex.stopExecutingtaskByVChannelName("mock")

		select {
		case <-mc.ctx.Done():
		default:
			t.FailNow()
		}
	})

}

func newMockCompactor(isvalid bool) *mockCompactor {
	ctx, cancel := context.WithCancel(context.TODO())
	return &mockCompactor{
		ctx:     ctx,
		cancel:  cancel,
		isvalid: isvalid,
	}
}

type mockCompactor struct {
	sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	isvalid       bool
	alwaysWorking bool
}

var _ compactor = (*mockCompactor)(nil)

func (mc *mockCompactor) compact() error {
	mc.Add(1)
	defer mc.Done()
	if !mc.isvalid {
		return errStart
	}
	if mc.alwaysWorking {
		<-mc.ctx.Done()
		return mc.ctx.Err()
	}
	return nil
}

func (mc *mockCompactor) getPlanID() UniqueID {
	return 1
}

func (mc *mockCompactor) stop() {
	if mc.cancel != nil {
		mc.cancel()
		mc.Wait()
	}
}

func (mc *mockCompactor) getCollection() UniqueID {
	return 1
}

func (mc *mockCompactor) getChannelName() string {
	return "mock"
}
