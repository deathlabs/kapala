# ---------------------------------------------------------
# Set default values.
# ---------------------------------------------------------

.DEFAULT_GOAL := build
VERSION := $(shell date +%Y-%m-%d-%H%M%S)
ORGANIZATION := deathlabs
APP := kapala

# Set scanner configuration and thresholds.
SEMGREP_CONFIG ?= auto
HADOLINT_FAILURE_THRESHOLD ?= warning
SBOM_PATH := $(APP)-sbom.json
VEX_YAML_PATH := vex.yaml
VEX_JSON_PATH := vex.json
GRYPE_FAILURE_THRESHOLD ?= medium

# ---------------------------------------------------------
# Update dependencies.
# ---------------------------------------------------------

.PHONY: update
.SILENT: update

update:
	echo "[*] Updating $(APP)'s dependencies"
	go get -u ./...
	go mod tidy

# ---------------------------------------------------------
# Check the repository for secrets.
# ---------------------------------------------------------

.PHONY: secrets
.SILENT: secrets
secrets:
	echo "[*] Checking $(APP)'s source code for hardcoded secrets"
	trufflehog filesystem \
		--no-update \
		--fail \
		--fail-on-scan-errors \
		--results=verified,unknown \
		--log-level=-1

# ---------------------------------------------------------
# Check for lint.
# ---------------------------------------------------------

.PHONY: lint
.SILENT: lint
lint:
	echo "[*] Checking $(APP)'s source code for lint"
	golangci-lint run

# ---------------------------------------------------------
# Check for first-party vulnerabilities.
# ---------------------------------------------------------

.PHONY: sast
.SILENT: sast
sast:
	echo "[*] Checking $(APP)'s source code for first-party vulnerabilities"
	semgrep scan --config "$(SEMGREP_CONFIG)"

# ---------------------------------------------------------
# Generate an SBOM.
# ---------------------------------------------------------

.PHONY: sbom
.SILENT: sbom
sbom: 
	echo "[*] Generating $(APP)'s SBOM"
	syft dir:. \
		--source-name $(APP) \
		--source-version $(VERSION) \
		-o cyclonedx-json="$(SBOM_PATH)"

# ---------------------------------------------------------
# Generate a VEX file.
# ---------------------------------------------------------

define VEX_FILTER
{
  "@context": "https://openvex.dev/ns/v0.2.0",
  "@id": strenv(VEX_ID),
  "author": strenv(VEX_AUTHOR),
  "timestamp": strenv(VEX_TIMESTAMP),
  "version": 1,
  "statements": [
    .advisories[] | {
      "vulnerability": {
        "name": .vulnerability
      },
      "products": [
        .products[] | {
          "@id": .
        }
      ],
      "status": .status,
      "justification": .justification,
      "impact_statement": .impact_statement
    }
  ]
}
endef

export VEX_FILTER

.PHONY: vex
.SILENT: vex
vex:
	echo "[*] Generating VEX statements for $(APP)"
	VEX_ID="$(VEX_ID_BASE)" \
	VEX_AUTHOR="$(VEX_AUTHOR)" \
	VEX_TIMESTAMP="$$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
	yq -o=json "$$VEX_FILTER" "$(VEX_YAML_PATH)" > "$(VEX_JSON_PATH)"

# ---------------------------------------------------------
# Update the vulnerability database.
# ---------------------------------------------------------

.PHONY: update-cve-db
.SILENT: update-cve-db
update-cve-db: 
	echo "[*] Updating the Grype vulnerability database"
	grype db update

# ---------------------------------------------------------
# Check for third-party vulnerabilities.
# ---------------------------------------------------------

.PHONY: dependency-scan
.SILENT: dependency-scan
dependency-scan: sbom vex update-cve-db
	echo "[*] Checking $(APP)'s SBOM for third-party vulnerabilities"
	grype sbom:"$(SBOM_PATH)" --vex "$(VEX_JSON_PATH)" --fail-on "$(GRYPE_FAILURE_THRESHOLD)"

# ---------------------------------------------------------
# Build the artifact.
# ---------------------------------------------------------

.PHONY: build
.SILENT: build

build: secrets lint sast dependency-scan
	echo "[*] Building and installing $(APP)"
	go install -ldflags="-s -w -X 'github.com/$(ORGANIZATION)/$(APP)/cmd.version=$(VERSION)'" . 
