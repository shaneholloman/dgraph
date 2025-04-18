/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package worker

import (
	"strings"

	"github.com/golang/glog"

	"github.com/hypermodeinc/dgraph/v25/protos/pb"
	"github.com/hypermodeinc/dgraph/v25/tok"
	"github.com/hypermodeinc/dgraph/v25/types"
	"github.com/hypermodeinc/dgraph/v25/x"
)

type matchFunc func(types.Val, *stringFilter) bool

type stringFilter struct {
	funcName string
	funcType FuncType
	lang     string
	tokens   []string
	match    matchFunc
	eqVals   []types.Val
	tokName  string
}

func matchStrings(uids *pb.List, values [][]types.Val, filter *stringFilter) *pb.List {
	rv := &pb.List{}
	if filter == nil {
		// Handle a nil filter as filtering all the elements out.
		return rv
	}

	for i := range values {
		for j := range values[i] {
			if filter.match(values[i][j], filter) {
				rv.Uids = append(rv.Uids, uids.Uids[i])
				break
			}
		}
	}

	return rv
}

func defaultMatch(value types.Val, filter *stringFilter) bool {
	tokenMap := map[string]bool{}
	for _, t := range filter.tokens {
		tokenMap[t] = false
	}

	tokens := tokenizeValue(value, filter)
	cnt := 0
	for _, token := range tokens {
		previous, ok := tokenMap[token]
		if ok {
			tokenMap[token] = true
			if !previous { // count only once
				cnt++
			}
		}
	}

	all := strings.HasPrefix(filter.funcName, "allof") // anyofterms or anyoftext

	if all {
		return cnt == len(filter.tokens)
	}
	return cnt > 0
}

func ineqMatch(value types.Val, filter *stringFilter) bool {
	if filter.funcName == eq {
		for _, v := range filter.eqVals {
			if types.CompareVals(filter.funcName, value, v) {
				return true
			}
		}
		return false
	} else if filter.funcName == between {
		return types.CompareVals("ge", value, filter.eqVals[0]) &&
			types.CompareVals("le", value, filter.eqVals[1])
	}

	return types.CompareVals(filter.funcName, value, filter.eqVals[0])
}

func tokenizeValue(value types.Val, filter *stringFilter) []string {
	tokenizer, found := tok.GetTokenizer(filter.tokName)
	// tokenizer was used in previous stages of query processing, it has to be available
	x.AssertTrue(found)

	tokens, err := tok.BuildTokens(value.Value, tok.GetTokenizerForLang(tokenizer, filter.lang))
	if err != nil {
		glog.Errorf("Error while building tokens: %s", err)
		return []string{}
	}
	return tokens
}
