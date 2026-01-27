#!/bin/bash
# Run integration tests in Docker with systemd support
set -e

CONTAINER_NAME="bak-integration-tests-$$"
IMAGE_NAME="bak-integration-test"

cleanup() {
    echo "Cleaning up..."
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

echo "Building test image..."
docker build -f Dockerfile.test -t "$IMAGE_NAME" .

echo "Starting test container with systemd..."
docker run -d \
    --name "$CONTAINER_NAME" \
    --privileged \
    -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --tmpfs /run \
    --tmpfs /run/lock \
    -e AUTO_RUN_TESTS=1 \
    "$IMAGE_NAME"

echo "Waiting for tests to complete..."
# Follow the test service logs
docker exec "$CONTAINER_NAME" bash -c '
    # Wait for systemd to be ready
    for i in {1..30}; do
        if systemctl is-system-running --wait 2>/dev/null; then
            break
        fi
        sleep 1
    done

    # Wait for tests to start and complete
    echo "Waiting for integration-tests service..."
    for i in {1..10}; do
        if systemctl is-active integration-tests.service 2>/dev/null || \
           systemctl show integration-tests.service --property=ActiveState 2>/dev/null | grep -q "inactive"; then
            break
        fi
        sleep 1
    done

    # Follow the journal for test output
    journalctl -u integration-tests.service -f --no-pager &
    JOURNAL_PID=$!

    # Wait for service to complete
    while systemctl is-active integration-tests.service 2>/dev/null; do
        sleep 1
    done

    kill $JOURNAL_PID 2>/dev/null || true

    # Get the exit code
    EXIT_CODE=$(systemctl show integration-tests.service --property=ExecMainStatus --value)
    echo ""
    echo "Test service exit code: $EXIT_CODE"
    exit ${EXIT_CODE:-1}
'

EXIT_CODE=$?
echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "Integration tests PASSED"
else
    echo "Integration tests FAILED (exit code: $EXIT_CODE)"
fi

exit $EXIT_CODE
