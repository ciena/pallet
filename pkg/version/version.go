/*
Copyright 2022 Ciena Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	nl = "\n"
)

// nolint:gochecknoglobals
var (
	version       string
	vcsURL        string
	vcsRef        string
	vcsCommitDate string
	vcsDirty      string
	goVersion     string
	os            string
	arch          string
	buildDate     string
)

// Spec defines the version information for the application.
type Spec struct {
	Version       string `json:"version,omitempty"`
	VcsURL        string `json:"vcsUrl,omitempty"`
	VcsRef        string `json:"vcsRef,omitempty"`
	VcsCommitDate string `json:"vcsCommitDate,omitempty"`
	VcsDirty      *bool  `json:"vcsDirty,omitempty"`
	GoVersion     string `json:"goVersion,omitempty"`
	OS            string `json:"os,omitempty"`
	Arch          string `json:"arch,omitempty"`
	BuildDate     string `json:"buildDate,omitempty"`
}

// Version returns the version information for the application.
func Version() *Spec {
	var dirty *bool

	if vcsDirty != "" {
		if parsed, err := strconv.ParseBool(vcsDirty); err == nil {
			dirty = &parsed
		}
	}

	return &Spec{
		Version:       version,
		VcsURL:        vcsURL,
		VcsRef:        vcsRef,
		VcsCommitDate: vcsCommitDate,
		VcsDirty:      dirty,
		GoVersion:     goVersion,
		OS:            os,
		Arch:          arch,
		BuildDate:     buildDate,
	}
}

// String returns the version information in string form.
// nolint:gocognit,cyclop
func (v *Spec) String() string {
	var ver strings.Builder

	if v.Version != "" {
		ver.WriteString(fmt.Sprintf("Version:       %s", v.Version))
	}

	if v.VcsURL != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("VcsURL:        %s", v.VcsURL))
	}

	if v.VcsRef != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("VcsRef:        %s", v.VcsRef))
	}

	if v.VcsCommitDate != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("VcsCommitDate: %s", v.VcsCommitDate))
	}

	if v.VcsDirty != nil {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("VcsDirty:      %t", *v.VcsDirty))
	}

	if v.GoVersion != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("GoVersion:     %s", v.GoVersion))
	}

	if v.OS != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("OS:            %s", v.OS))
	}

	if v.Arch != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("Arch:          %s", v.Arch))
	}

	if v.BuildDate != "" {
		if ver.Len() != 0 {
			ver.WriteString(nl)
		}

		ver.WriteString(fmt.Sprintf("BuildDate:     %s", v.BuildDate))
	}

	return ver.String()
}
