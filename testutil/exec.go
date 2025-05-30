/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package testutil

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/hypermodeinc/dgraph/v25/x"
)

// These are exported so they can also be set directly from outside this package.
var (
	ShowOutput  = os.Getenv("DEBUG_SHOW_OUTPUT") != ""
	ShowError   = os.Getenv("DEBUG_SHOW_ERROR") != ""
	ShowCommand = os.Getenv("DEBUG_SHOW_COMMAND") != ""
)

// CmdOpts sets the options to run a single command.
type CmdOpts struct {
	Dir string
}

// Exec runs a single external command.
func Exec(argv ...string) error {
	_, err := Pipeline([][]string{argv})
	return err
}

// ExecWithOpts runs a single external command with the given options.
func ExecWithOpts(argv []string, opts CmdOpts) error {
	_, err := pipelineInternal([][]string{argv}, []CmdOpts{opts})
	return err
}

// Pipeline runs several commands such that the output of one command becomes the input of the next.
// The first argument should be an two-dimensional array containing the commands.
// TODO: allow capturing output, sending to terminal, etc
func Pipeline(cmds [][]string) (string, error) {
	return pipelineInternal(cmds, nil)
}

// piplineInternal takes a list of commands and a list of options (one for each).
// If opts is nil, all commands should be run with the default options.
func pipelineInternal(cmds [][]string, opts []CmdOpts) (string, error) {
	x.AssertTrue(opts == nil || len(cmds) == len(opts))

	var p io.ReadCloser
	var numCmds = len(cmds)

	cmd := make([]*exec.Cmd, numCmds)
	var stdout bytes.Buffer

	// Run all commands in parallel, connecting stdin of each to the stdout of the previous.
	for i, c := range cmds {
		lastCmd := i == numCmds-1
		if ShowCommand {
			fmt.Fprintf(os.Stderr, "%+v", c)
		}

		cmd[i] = exec.Command(c[0], c[1:]...)
		cmd[i].Stdin = p

		if opts != nil {
			cmd[i].Dir = opts[i].Dir
		}

		if !lastCmd {
			p, _ = cmd[i].StdoutPipe()
		} else {
			cmd[i].Stdout = &stdout
		}

		if ShowOutput {
			cmd[i].Stderr = os.Stderr
		} else if ShowError {
			cmd[i].Stderr = os.Stderr
		}

		if ShowCommand {
			if lastCmd {
				fmt.Fprintf(os.Stderr, "\n")
			} else {
				fmt.Fprintf(os.Stderr, "\n| ")
			}
		}

		err := cmd[i].Start()
		x.Check(err)
	}

	// Make sure to properly reap all spawned processes, but only save the error from the
	// earliest stage of the pipeline.
	var err error
	for i := range cmds {
		e := cmd[i].Wait()
		if e != nil && err == nil {
			err = e
		}
	}

	outStr := stdout.String()
	if ShowOutput {
		fmt.Println(outStr)
	}

	return outStr, err
}

func DgraphBinaryPath() string {
	// Useful for OSX, as $GOPATH/bin/dgraph is set to the linux binary for docker
	if dgraphBinary := os.Getenv("DGRAPH_BINARY"); dgraphBinary != "" {
		return dgraphBinary
	}

	gopath := os.Getenv("GOPATH")

	if gopath == "" {
		gopath = build.Default.GOPATH
	}

	return os.ExpandEnv(gopath + "/bin/dgraph")
}

func DetectRaceInZeros(prefix string) bool {
	for i := 0; i <= 3; i++ {
		in := GetContainerInstance(prefix, "zero"+strconv.Itoa(i))
		if DetectIfRaceViolation(in) {
			return true
		}
	}
	return false
}

func DetectRaceInAlphas(prefix string) bool {
	for i := 0; i <= 6; i++ {
		in := GetContainerInstance(prefix, "alpha"+strconv.Itoa(i))
		if DetectIfRaceViolation(in) {
			return true
		}
	}
	return false
}

func DetectIfRaceViolation(instance ContainerInstance) bool {
	c := instance.GetContainer()
	if c == nil {
		return false
	}

	logCmd := exec.Command("docker", "logs", c.ID)
	out, err := logCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: while getting docker logs %v\n", err)
		return false
	}

	return CheckIfRace(out)
}

func CheckIfRace(output []byte) bool {
	awkCmd := exec.Command("awk", "/WARNING: DATA RACE/{flag=1}flag;/==================/{flag=0}")
	awkCmd.Stdin = bytes.NewReader(output)
	out, err := awkCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: while getting race content %v\n", err)
		return false
	}

	if len(out) > 0 {
		fmt.Printf("DATA RACE DETECTED %s\n", string(out))
		return true
	}
	return false
}
