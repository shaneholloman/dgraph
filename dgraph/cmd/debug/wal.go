/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package debug

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"google.golang.org/protobuf/proto"

	"github.com/hypermodeinc/dgraph/v25/protos/pb"
	"github.com/hypermodeinc/dgraph/v25/raftwal"
	"github.com/hypermodeinc/dgraph/v25/x"
)

func printEntry(es raftpb.Entry, pending map[uint64]bool, isZero bool) {
	var buf bytes.Buffer
	defer func() {
		fmt.Printf("%s\n", buf.Bytes())
	}()

	var key uint64
	if len(es.Data) >= 8 {
		key = binary.BigEndian.Uint64(es.Data[:8])
	}
	fmt.Fprintf(&buf, "%d . %d . %v . %-6s . %8d .", es.Term, es.Index, es.Type,
		humanize.Bytes(uint64(es.Size())), key)
	if es.Type == raftpb.EntryConfChange {
		return
	}
	if len(es.Data) == 0 {
		return
	}
	var err error
	if isZero {
		var zpr pb.ZeroProposal
		if err = proto.Unmarshal(es.Data[8:], &zpr); err == nil {
			printZeroProposal(&buf, &zpr)
			return
		}
	} else {
		var pr pb.Proposal
		if err = proto.Unmarshal(es.Data[8:], &pr); err == nil {
			printAlphaProposal(&buf, &pr, pending)
			return
		}
	}
	fmt.Fprintf(&buf, " Unable to parse Proposal: %v", err)
}

type RaftStore interface {
	raft.Storage
	Checkpoint() (uint64, error)
	HardState() (raftpb.HardState, error)
}

func printBasic(store RaftStore) (uint64, uint64) {
	fmt.Println()
	snap, err := store.Snapshot()
	if err != nil {
		fmt.Printf("Got error while retrieving snapshot: %v\n", err)
	} else {
		fmt.Printf("Snapshot Metadata: %+v\n", snap.Metadata)
		var ds pb.Snapshot
		var zs pb.ZeroSnapshot
		if err := proto.Unmarshal(snap.Data, &ds); err == nil {
			fmt.Printf("Snapshot Alpha: %+v\n", ds)
		} else if err := proto.Unmarshal(snap.Data, &zs); err == nil {
			for gid, group := range zs.State.GetGroups() {
				fmt.Printf("\nGROUP: %d\n", gid)
				for _, member := range group.GetMembers() {
					fmt.Printf("Member: %+v .\n", member)
				}
				for _, tablet := range group.GetTablets() {
					fmt.Printf("Tablet: %+v .\n", tablet)
				}
				group.Members = nil
				group.Tablets = nil
				fmt.Printf("Group: %d %+v .\n", gid, group)
			}
			zs.State.Groups = nil
			fmt.Printf("\nSnapshot Zero: %+v\n", zs)
		} else {
			fmt.Printf("Unable to unmarshal Dgraph snapshot: %v", err)
		}
	}
	fmt.Println()

	if hs, err := store.HardState(); err != nil {
		fmt.Printf("Got error while retrieving hardstate: %v\n", err)
	} else {
		fmt.Printf("Hardstate: %+v\n", hs)
	}

	if chk, err := store.Checkpoint(); err != nil {
		fmt.Printf("Got error while retrieving checkpoint: %v\n", err)
	} else {
		fmt.Printf("Checkpoint: %d\n", chk)
	}

	lastIdx, err := store.LastIndex()
	if err != nil {
		fmt.Printf("Got error while retrieving last index: %v\n", err)
	}
	startIdx := snap.Metadata.Index + 1
	fmt.Printf("Last Index: %d . Num Entries: %d .\n\n", lastIdx, lastIdx-startIdx)
	return startIdx, lastIdx
}

func printRaft(store *raftwal.DiskStorage) {
	isZero := store.Uint(raftwal.GroupId) == 0

	pending := make(map[uint64]bool)
	startIdx, lastIdx := printBasic(store)

	for startIdx < lastIdx {
		entries, err := store.Entries(startIdx, lastIdx+1, 64<<20)
		x.Check(err)
		for _, ent := range entries {
			printEntry(ent, pending, isZero)
			startIdx = x.Max(startIdx, ent.Index)
		}
		startIdx = startIdx + 1
	}
}

func overwriteSnapshot(store *raftwal.DiskStorage) error {
	snap, err := store.Snapshot()
	x.Checkf(err, "Unable to get snapshot")
	cs := snap.Metadata.ConfState
	fmt.Printf("Confstate: %+v\n", cs)

	var dsnap pb.Snapshot
	if len(snap.Data) > 0 {
		x.Check(proto.Unmarshal(snap.Data, &dsnap))
	}
	fmt.Printf("Previous snapshot: %+v\n", dsnap)

	splits := strings.Split(opt.wsetSnapshot, ",")
	x.AssertTruef(len(splits) == 3,
		"Expected term,index,readts in string. Got: %s", splits)
	term, err := strconv.Atoi(splits[0])
	x.Check(err)
	index, err := strconv.Atoi(splits[1])
	x.Check(err)
	readTs, err := strconv.Atoi(splits[2])
	x.Check(err)

	ent := raftpb.Entry{
		Term:  uint64(term),
		Index: uint64(index),
		Type:  raftpb.EntryNormal,
	}
	fmt.Printf("Using term: %d , index: %d , readTs : %d\n", term, index, readTs)
	if dsnap.Index >= ent.Index {
		fmt.Printf("Older snapshot is >= index %d", ent.Index)
		return nil
	}

	// We need to write the Raft entry first.
	fmt.Printf("Setting entry: %+v\n", ent)
	hs := raftpb.HardState{
		Term:   ent.Term,
		Commit: ent.Index,
	}
	fmt.Printf("Setting hard state: %+v\n", hs)
	err = store.Save(&hs, []raftpb.Entry{ent}, &snap)
	x.Check(err)

	dsnap.Index = ent.Index
	dsnap.ReadTs = uint64(readTs)

	fmt.Printf("Setting snapshot to: %+v\n", dsnap)
	data, err := proto.Marshal(&dsnap)
	x.Check(err)
	if err = store.CreateSnapshot(dsnap.Index, &cs, data); err != nil {
		fmt.Printf("Created snapshot with error: %v\n", err)
	}
	return err
}

func handleWal(store *raftwal.DiskStorage) error {
	rid := store.Uint(raftwal.RaftId)
	gid := store.Uint(raftwal.GroupId)

	fmt.Printf("Raft Id = %d Groupd Id = %d\n", rid, gid)
	switch {
	case len(opt.wsetSnapshot) > 0:
		return overwriteSnapshot(store)
	case opt.wtruncateUntil != 0:
		store.TruncateEntriesUntil(opt.wtruncateUntil)
	default:
		printRaft(store)
	}
	return nil
}
