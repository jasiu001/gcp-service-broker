// Copyright 2018 the Service Broker Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"bytes"
	"log"
	"text/template"

	"github.com/GoogleCloudPlatform/gcp-service-broker/pkg/config/migration"
	"github.com/GoogleCloudPlatform/gcp-service-broker/utils"
)

const (
	appName         = "gcp-service-broker"
	appDescription  = "A service broker for Google Cloud Platform services."
	stemcellOs      = "ubuntu-xenial"
	stemcellVersion = "170.82"

	buildpack     = "go_buildpack"
	goPackageName = "github.com/GoogleCloudPlatform/gcp-service-broker"
	goVersion     = "go1.12"

	copyrightHeader = `# Copyright the Service Broker Project Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file is AUTOGENERATED by ./gcp-service-broker generate, DO NOT EDIT IT.

---`
	manifestYmlTemplate = copyrightHeader + `
applications:
- name: {{.appName}}
  memory: 1G
  buildpacks:
  - {{.buildpack}}
  env:
    GOPACKAGENAME: {{.goPackageName}}
    GOVERSION: {{.goVersion}}`
	tileYmlTemplate = copyrightHeader + `
name: {{.appName}}
icon_file: gcp_logo.png
label: Google Cloud Platform Service Broker
description: '{{.appDescription}}'
product_version: "{{.appVersion}}"
org: system

stemcell_criteria:
  os: '{{.stemcellOs}}'
  version: '{{.stemcellVersion}}'

apply_open_security_group: true

migration: |
{{.migrationScript}}

packages:
- name: {{.appName}}
  type: app-broker
  manifest:
    buildpacks:
    - {{.buildpack}}
    path: /tmp/gcp-service-broker.zip
    env:
      GOPACKAGENAME: {{.goPackageName}}
      GOVERSION: {{.goVersion}}
      # You can override plans here.
  needs_cf_credentials: true
  enable_global_access_to_plans: true


# Uncomment this section if you want to display forms with configurable
# properties in Ops Manager. These properties will be passed to your
# applications as environment variables. You can also refer to them
# elsewhere in this template by using:
#     (( .properties.<property-name> ))
`
)

// GenerateManifest creates a manifest.yml from a template.
func GenerateManifest() string {
	return runPcfTemplate(manifestYmlTemplate)
}

// GenerateTile creates a tile.yml from a template.
func GenerateTile() string {
	return runPcfTemplate(tileYmlTemplate) + GenerateFormsString()
}

func runPcfTemplate(templateString string) string {
	migrator := migration.FullMigration()

	vars := map[string]interface{}{
		"appName":         appName,
		"appVersion":      utils.Version,
		"appDescription":  appDescription,
		"buildpack":       buildpack,
		"goPackageName":   goPackageName,
		"goVersion":       goVersion,
		"stemcellOs":      stemcellOs,
		"stemcellVersion": stemcellVersion,
		"migrationScript": utils.Indent(migrator.TileScript, "  "),
	}

	tmpl, err := template.New("tmpl").Parse(templateString)
	if err != nil {
		log.Fatalf("parsing: %s", err)
	}

	// Run the template to verify the output.
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		log.Fatalf("execution: %s", err)
	}

	return buf.String()
}
