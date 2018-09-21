// Copyright 2018 Pivotal Software, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	flag "github.com/spf13/pflag"
)

const (
	childEnv = "CERTSWAP_CHILD"
	pathEnv  = "CERTSWAP_PATH"
)

var cas = flag.StringArray("ca", nil, "path to a CA PEM file which should be in the system cert pool")

func main() {
	flag.Parse()

	if inParent() {
		dir, err := buildCAPool(*cas)
		if err != nil {
			log.Fatal(err)
		}

		// We don't want to pollute the system path while this command runs so
		// we enter a new mount namespace. Unfortunately, we can't enter a new
		// mount namespace without being root so we create a new user namespace
		// where we're root at the same time. We then re-exec the current
		// binary in those namespaces.
		status := execInMountNS(envvar(childEnv, "true"), envvar(pathEnv, dir))
		if err := os.RemoveAll(dir); err != nil {
			log.Println("failed to cleanup temporary directory:", err)
		}
		os.Exit(status)
	}

	// Mount over the system cert path with our certificate directory.
	if err := syscall.Mount(os.Getenv(pathEnv), "/etc/ssl/certs", "", syscall.MS_BIND|syscall.MS_RDONLY, ""); err != nil {
		log.Fatal(err)
	}

	if len(flag.Args()) == 0 {
		usage()
	}

	os.Unsetenv(childEnv)
	os.Unsetenv(pathEnv)

	exe := flag.Arg(0)
	args := flag.Args()[1:]

	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	os.Exit(runCommand(cmd))
}

func execInMountNS(env ...string) int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("/proc/self/exe", os.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
		GidMappingsEnableSetgroups: false,
	}
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return runCommand(cmd)
}

func usage() {
	fmt.Println("usage: certswap [--ca PATH]... -- COMMAND [ARGUMENTS...]")
	os.Exit(2)
}

func runCommand(cmd *exec.Cmd) int {
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			if ws, ok := exit.Sys().(syscall.WaitStatus); ok {
				return ws.ExitStatus()
			}
			return -1
		}
		log.Println("failed to run child:", err.Error())
		return -1
	}
	return 0
}

// InChild returns true if we are in the user-namespaced inner execution.
func inChild() bool {
	return os.Getenv(childEnv) == "true"
}

// InParent returns true if we are in the unprivileged outer execution.
func inParent() bool {
	return !inChild()
}

func buildCAPool(cas []string) (string, error) {
	dir, err := ioutil.TempDir("", "certswap")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	for _, ca := range cas {
		bs, err := ioutil.ReadFile(ca)
		if err != nil {
			return "", err
		}
		if _, err := buf.Write(bs); err != nil {
			return "", err
		}
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "ca-certificates.crt"), buf.Bytes(), 0644); err != nil {
		return "", err
	}

	return dir, nil
}

func envvar(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
