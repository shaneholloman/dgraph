/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

func Tokenizer() interface{} { return FactorTokenizer{} }

type FactorTokenizer struct{}

func (FactorTokenizer) Name() string     { return "factor" }
func (FactorTokenizer) Type() string     { return "int" }
func (FactorTokenizer) Identifier() byte { return 0xfe }

func (FactorTokenizer) Tokens(value interface{}) ([]string, error) {
	x := value.(int64)
	if x <= 1 {
		return nil, errors.Errorf("Cannot factor int <= 1: %d", x)
	}
	var toks []string
	for p := int64(2); x > 1; p++ {
		if x%p == 0 {
			toks = append(toks, encodeInt(p))
			for x%p == 0 {
				x /= p
			}
		}
	}
	return toks, nil

}

func encodeInt(x int64) string {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutVarint(buf[:], x)
	return string(buf[:n])
}
