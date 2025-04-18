/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package testutil

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func GeneratePlugins(raceEnabled bool) {
	_, curr, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Print("error while getting current file")
		return
	}
	var soFiles []string
	for i, src := range []string{
		"./custom_plugins/anagram/main.go",
		"./custom_plugins/cidr/main.go",
		"./custom_plugins/factor/main.go",
		"./custom_plugins/rune/main.go",
	} {
		so := "./custom_plugins/" + strconv.Itoa(i) + ".so"
		fmt.Printf("compiling plugin: src=%q so=%q\n", src, so)
		opts := []string{"build"}
		if raceEnabled {
			opts = append(opts, "-race")
		}
		opts = append(opts, "-buildmode=plugin", "-o", so, src)
		cmd := exec.Command("go", opts...)
		cmd.Dir = filepath.Dir(curr)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Output: %v\n", string(out))
			return
		}
		absSO, err := filepath.Abs(so)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		soFiles = append(soFiles, absSO)
	}

	fmt.Printf("plugin build completed. Files are: %s\n", strings.Join(soFiles, ","))
}
