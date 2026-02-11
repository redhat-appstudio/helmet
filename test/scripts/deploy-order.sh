#!/usr/bin/env bash
#
# deploy-order.sh records the Helm release name into a "deploy-sequence"
# ConfigMap, appending to the existing sequence. Uses optimistic concurrency
# control (resourceVersion) to handle parallel deployments safely.
#
# Required environment variables:
#   RELEASE_NAME - the Helm release name to record
#   NAMESPACE    - the Kubernetes namespace
#

set -euo pipefail

CM_NAME="deploy-sequence"

# Ensure the ConfigMap exists; ignore AlreadyExists errors.
kubectl create configmap "$CM_NAME" \
    --from-literal=sequence="" \
    --namespace="$NAMESPACE" 2>/dev/null || true

# Optimistic-concurrency append: read the current sequence and resourceVersion,
# build the updated value, then patch with the resourceVersion so the API server
# rejects stale writes. On conflict the retry loop re-reads and tries again.
for attempt in $(seq 1 5); do
    # Single read to avoid TOCTOU: resourceVersion and sequence are fetched
    # atomically. Tab delimiter is safe — neither field contains tabs.
    READ=$(kubectl get configmap "$CM_NAME" \
        --namespace="$NAMESPACE" \
        -o go-template='{{.metadata.resourceVersion}}{{"\t"}}{{index .data "sequence"}}')
    RV="${READ%%	*}"
    CURRENT="${READ#*	}"

    if [[ -z "$CURRENT" ]]; then
        UPDATED="$RELEASE_NAME"
    else
        UPDATED="$(printf '%s\n%s' "$CURRENT" "$RELEASE_NAME")"
    fi

    # Escape newlines for valid JSON embedding. The API server decodes JSON \n
    # back to actual newlines in the stored value.
    ESCAPED=$(printf '%s' "$UPDATED" | sed ':a;N;$!ba;s/\n/\\n/g')

    # Patch with resourceVersion for optimistic locking. The API server returns
    # 409 Conflict if the resourceVersion has changed since the read, causing
    # kubectl to exit non-zero.
    PAYLOAD="{\"metadata\":{\"resourceVersion\":\"${RV}\"},\"data\":{\"sequence\":\"${ESCAPED}\"}}"

    if kubectl patch configmap "$CM_NAME" \
        --namespace="$NAMESPACE" \
        --type=merge \
        -p "$PAYLOAD"; then
        echo "Recorded deploy order: $RELEASE_NAME (attempt $attempt)"
        exit 0
    fi

    echo "Conflict on attempt $attempt, retrying..."
    sleep 1
done

echo "ERROR: failed to record deploy order after 5 attempts"
exit 1
