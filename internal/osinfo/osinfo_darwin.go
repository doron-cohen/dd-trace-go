// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package osinfo

import (
	"os/exec"
	"runtime"
	"strings"
)

DARWIN

func osName() string {
	return runtime.GOOS
}

func osVersion() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "unknown"
	}
	return strings.Trim(string(out), "\n")
}
