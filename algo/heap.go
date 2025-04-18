/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package algo

import (
	"github.com/hypermodeinc/dgraph/v25/codec"
)

type elem struct {
	val     uint64 // Value of this element.
	listIdx int    // Which list this element comes from.

	// The following fields are only used when merging pb.UidPack objects.
	decoder   *codec.Decoder // pointer to the decoder for this pb.UidPack object.
	blockIdx  int            // The current position in the current pb.UidBlock object.
	blockUids []uint64       // The current block.
}

type uint64Heap []elem

func (h uint64Heap) Len() int           { return len(h) }
func (h uint64Heap) Less(i, j int) bool { return h[i].val < h[j].val }
func (h uint64Heap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *uint64Heap) Push(x interface{}) {
	*h = append(*h, x.(elem))
}

func (h *uint64Heap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
