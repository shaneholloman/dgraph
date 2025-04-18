/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidBlockErr(t *testing.T) {
	query := `
query {
  me(func: eq(age, 34)) {
    uid
    friend {
      uid
      age
    }
  }
}`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestExtraRightCurlErr(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) {
      uid
      friend {
        uid
        age
      }
    }
  }
}
}
`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Too many right curl")
}

func TestNoMutationErr(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) {
      uid
      friend {
        uid age
      }
    }
  }
}
`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Empty mutation block")
}

func TestMultipleQueryErr(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) {
      uid
      friend {
        uid
        age
      }
    }
  }

  query {
    me2(func: eq(age, 34)) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Multiple query ops inside upsert block")
}

func TestEmptyUpsertErr(t *testing.T) {
	query := `upsert {}`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Empty mutation block")
}

func TestNoRightCurlErr(t *testing.T) {
	query := `upsert {`
	_, err := ParseDQL(query)
	require.Contains(t, err.Error(), "Unclosed upsert block")
}

func TestIncompleteBlockErr(t *testing.T) {
	query := `
upsert {
  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }

  query {
    me(func: eq(age, "{
`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unexpected end of input")
}

func TestMissingQueryErr(t *testing.T) {
	query := `
upsert {
  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Query op not found in upsert block")
}

func TestUpsertWithFragment(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) {
      ...fragmentA
      friend {
        ...fragmentA
        age
      }
    }
  }

  fragment fragmentA {
    uid
  }

  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestUpsertEx1(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, "{")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestUpsertWithSpaces(t *testing.T) {
	query := `
upsert

{
  query

  {
    me(func: eq(age, "{")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation

  {
    set
    {
      "_:user1" <age> "45" .

      # This is a comment
      "_:user1" <name> "{vishesh" .
    }}
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestUpsertWithBlankNode(t *testing.T) {
	query := `
upsert {
  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }

  query {
    me(func: eq(age, 34)) {
      uid
      friend {
        uid
        age
      }
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestUpsertMutationThenQuery(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation {
    set {
      "_:user1" <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestUpsertWithFilter(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation {
    set {
      uid(a) <age> "45"
      uid(b) <age> "45" .
    }
  }
}`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestConditionalUpsertWithNewlines(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      m as uid
      friend {
        f as uid
        age
      }
    }
  }

  mutation @if(eq(len(m), 1)
               AND
               gt(len(f), 0)) {
    set {
      uid(m) <age> "45" .
      uid(f) <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestConditionalUpsertFuncTree(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation @if( ( eq(len(m), 1)
                  OR
                  lt(90, len(h)))
                AND
                gt(len(f), 0)) {
    set {
      uid(m) <age> "45" .
      uid(f) <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestConditionalUpsertMultipleFuncArg(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation @if( ( eq(len(m), len(t))
                  OR
                  lt(90, len(h)))
                AND
                gt(len(f), 0)) {
    set {
      uid(m) <age> "45" .
      uid(f) <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.NoError(t, err)
}

func TestConditionalUpsertErrMissingRightRound(t *testing.T) {
	query := `
upsert {
  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
      friend {
        uid
        age
      }
    }
  }

  mutation @if(eq(len(m, 1)
               AND
               gt(len(f), 0)) {
    set {
      uid(m) <age> "45" .
      uid(f) <age> "45" .
    }
  }
}
`
	_, err := ParseDQL(query)
	require.Contains(t, err.Error(), "Matching brackets not found")
}

func TestConditionalUpsertErrUnclosed(t *testing.T) {
	query := `upsert {
  mutation @if(eq(len(m), 1) AND gt(len(f), 0))`
	_, err := ParseDQL(query)
	require.Contains(t, err.Error(), "Unclosed mutation action")
}

func TestConditionalUpsertErrInvalidIf(t *testing.T) {
	query := `upsert {
  mutation @if`
	_, err := ParseDQL(query)
	require.Contains(t, err.Error(), "Matching brackets not found")
}

func TestConditionalUpsertErrWrongIf(t *testing.T) {
	query := `upsert {
  mutation @fi( ( eq(len(m), 1)
                  OR
                  lt(len(h), 90))
                AND
                gt(len(f), 0)) {
    set {
      uid(m) <age> "45" .
      uid(f) <age> "45" .
    }
  }

  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
      friend {
        uid
        age
      }
    }
  }
}
`
	_, err := ParseDQL(query)
	require.Contains(t, err.Error(), "Expected @if, found [@fi]")
}

func TestMultipleMutation(t *testing.T) {
	query := `
upsert {
  mutation @if(eq(len(m), 1)) {
    set {
      uid(m) <age> "45" .
    }
  }

  mutation @if(not(eq(len(m), 1))) {
    set {
      uid(f) <age> "45" .
    }
  }

  mutation {
    set {
      _:user <age> "45" .
    }
  }

  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
    }
  }
}`
	req, err := ParseDQL(query)
	require.NoError(t, err)
	require.Equal(t, 3, len(req.Mutations))
}

func TestMultipleMutationDifferentOrder(t *testing.T) {
	query := `
upsert {
  mutation @if(eq(len(m), 1)) {
    set {
      uid(m) <age> "45" .
    }
  }

  query {
    me(func: eq(age, 34)) @filter(ge(name, "user")) {
      uid
    }
  }

  mutation @if(not(eq(len(m), 1))) {
    set {
      uid(f) <age> "45" .
    }
  }

  mutation {
    set {
      _:user <age> "45" .
    }
  }
}`
	req, err := ParseDQL(query)
	require.NoError(t, err)
	require.Equal(t, 3, len(req.Mutations))
}
