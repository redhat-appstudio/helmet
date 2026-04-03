#!/bin/sh

# Bumps the all the go direct dependencies one by one,
# ignoring versions that breaks the build.

set -eu

SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null
    pwd
)"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/}
" >&2
}

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
        -d | --debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: $1"
            usage
            exit 1
            ;;
        esac
        shift
    done
}

cleanup() {
    rm -rf vendor/
    [ -n "${depsfile:-}" ] && rm -f "$depsfile"
    git restore .
}

exit_cleanup() {
    cleanup
    if [ -n "$CONTAINER_NAME" ]; then
        podman rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
}

init() {
    trap exit_cleanup EXIT

    PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
    cd "$PROJECT_DIR"

    GO_IMAGE="$(grep "/go-toolset:" "$PROJECT_DIR/example/helmet-ex/Dockerfile" | cut -d" " -f2)"
    if ! podman pull "$GO_IMAGE"; then
        echo "[ERROR] \`podman pull $GO_IMAGE\` failed"
        exit 1
    fi

    CONTAINER_NAME="helmet-update-go-deps-$$"
    if ! podman run -d \
        --name "$CONTAINER_NAME" \
        --volume "$PROJECT_DIR":/app \
        --workdir /app \
        "$GO_IMAGE" \
        sleep infinity; then
        echo "[ERROR] failed to start helper container"
        exit 1
    fi
}

run() {
    podman exec --workdir /app "$CONTAINER_NAME" "$@"
}

update_dependency() {
    echo "# $DEPENDENCY"

    if ! run go get -u "$DEPENDENCY"; then
        echo "[ERROR] \`go get -u $DEPENDENCY\` failed"
        cleanup
        return
    fi
    run go mod verify
    if ! run go mod tidy -v; then
        echo "[ERROR] \`go mod tidy\` failed"
        cleanup
        return
    fi

    if git diff --exit-code --quiet; then
        echo "No update"
        return
    fi

    run go mod vendor
    if ! run make lint; then
        echo "[ERROR] \`make lint\` failed"
        cleanup
        return
    fi

    if run make; then
        git add .
        git commit -m "chore: bump go dependency $DEPENDENCY"
    else
        echo "[ERROR] \`make\` failed"
        cleanup
    fi
}

action() {
    init
    depsfile=$(mktemp "${TMPDIR:-/tmp}/helmet-update-deps.XXXXXX")
    if ! go list -mod=readonly -f '{{.Path}}' -m all >"$depsfile"; then
        exit 1
    fi
    while IFS= read -r DEPENDENCY || [ -n "$DEPENDENCY" ]; do
        if ! grep -qE "[[:space:]]${DEPENDENCY}[[:space:]]" go.mod; then
            continue
        fi
        echo
        update_dependency
        echo
    done <"$depsfile"
    rm -f "$depsfile"
}

main() {
    parse_args "$@"
    action
}

main "$@"
