#!/usr/bin/env bash
#
# Wrapper around "govulncheck" that filters stdlib/toolchain findings.
# 
# Runs govulncheck in JSON mode and filters out vulnerabilities in modules
# specified by "GOVULNCHECK_IGNORE_MODULES".
#
# Exit codes:
#   0 = clean (or only ignored findings)
#   1 = actionable (non-ignored) vulnerabilities found
#   2 = prerequisite failure (jq not found)
#

set -eu -o pipefail

# Space-separated module names to suppress.
export GOVULNCHECK_IGNORE_MODULES="${GOVULNCHECK_IGNORE_MODULES:-}"
# Space-separated vulnerability IDs to suppress
export GOVULNCHECK_IGNORE_IDS="${GOVULNCHECK_IGNORE_IDS:-}"
# Package pattern to scan (default: "./...").
export GOVULNCHECK_PACKAGES="${GOVULNCHECK_PACKAGES:-./...}"

# Check prerequisites.
check_prerequisites() {
    if ! command -v jq &>/dev/null; then
        echo "Error: jq is required but not installed." >&2
        exit 2
    fi
}

# Pin Go toolchain to the version from "go.mod". Using "GOTOOLCHAIN=auto" only
# upgrades, never downgrades. Force exact version.
pin_go_toolchain() {
    local go_version
    go_version=$(go mod edit -json | jq -r '.Go')
    if [[ -z "${go_version}" ]]; then
        echo "Error: Failed to read Go version from go.mod" >&2
        exit 2
    fi
    export GOTOOLCHAIN="go${go_version}"
    echo "${go_version}"
}

# Run govulncheck in JSON mode and return the output.
run_govulncheck() {
    go tool govulncheck -format json "${GOVULNCHECK_PACKAGES}" 2>&1
}

# Filter findings using jq to remove ignored modules and IDs.
filter_findings() {
    local json_output="${1}"

    echo "${json_output}" | jq -s \
        --arg ignore_modules "${GOVULNCHECK_IGNORE_MODULES}" \
        --arg ignore_ids "${GOVULNCHECK_IGNORE_IDS}" '
        [.[] | select(.finding != null) | .finding] |
        ($ignore_modules | split(" ") | map(select(. != ""))) as $ignored_mods |
        ($ignore_ids | split(" ") | map(select(. != ""))) as $ignored_ids |
        [.[] |
            select((.trace[0].module // "") | IN($ignored_mods[]) | not) |
            select(.osv | IN($ignored_ids[]) | not)
        ] |
        unique_by(.osv)
    '
}

# Report findings to the user.
report_findings() {
    local filtered_findings="${1}"
    local vuln_count

    vuln_count=$(echo "${filtered_findings}" | jq 'length')

    if [[ "${vuln_count}" -eq 0 ]]; then
        echo "No actionable vulnerabilities found."
        return 0
    fi

    echo "Found '${vuln_count}' actionable item(s):"
    echo ""

    echo "${filtered_findings}" | jq -r '.[] |
        "- \(.osv) in \(.trace[0].module)" +
        if .trace[0].package then "/\(.trace[0].package)" else "" end +
        if .trace[0].version then " (current: \(.trace[0].version))" else "" end +
        if .fixed_version then ", fixed in: \(.fixed_version)" else "" end +
        "\n  https://pkg.go.dev/vuln/\(.osv)"
    '

    echo ""
    echo "Run 'GOVULNCHECK_IGNORE_MODULES=\"\" GOVULNCHECK_IGNORE_IDS=\"\"" \
        "${0}' to disable filtering."

    return 1
}

main() {
    check_prerequisites

    local go_version
    go_version=$(pin_go_toolchain)

    echo "Running govulncheck with Go '${go_version}'..."
    echo ""
    echo "   Ignoring modules: '${GOVULNCHECK_IGNORE_MODULES:-<none>}'"
    echo "       Ignoring IDs: '${GOVULNCHECK_IGNORE_IDS:-<none>}'"
    echo "  Scanning packages: '${GOVULNCHECK_PACKAGES}'"
    echo ""

    local json_output
    json_output=$(run_govulncheck)

    local filtered_findings
    filtered_findings=$(filter_findings "${json_output}")

    report_findings "$filtered_findings"
}

main "$@"
