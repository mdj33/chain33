// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"
	"time"

	"github.com/33cn/chain33/common/db/mocks"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSaveFee(t *testing.T) {
	kvdb := &mocks.KVDB{}
	ctx := &executorCtx{
		stateHash:  nil,
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: []byte("hash"),
	}
	executor := newExecutor(ctx, &Executor{}, kvdb, nil, nil)
	kvdb.On("Get", mock.Anything).Return([]byte("fees"), nil)
	kv, err := saveFee(executor, &types.TotalFee{}, []byte("parenthash"), []byte("hash"))
	assert.Nil(t, kv)
	assert.NotNil(t, err)

}
