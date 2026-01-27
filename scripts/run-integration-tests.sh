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
    --cgroupns=host \
    -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
    --tmpfs /run \
    --tmpfs /run/lock \
    "$IMAGE_NAME"

echo "Waiting for systemd to be ready..."
SYSTEMD_READY=false
for i in {1..60}; do
    # Check if container is still running
    if ! docker ps -q -f "name=$CONTAINER_NAME" | grep -q .; then
        echo "ERROR: Container exited unexpectedly"
        echo "=== Container logs ==="
        docker logs "$CONTAINER_NAME" 2>&1 || true
        echo "=== Container inspect ==="
        docker inspect "$CONTAINER_NAME" --format='{{.State.Status}} - Exit: {{.State.ExitCode}} - Error: {{.State.Error}}' 2>&1 || true
        exit 1
    fi

    # Check if systemd is ready
    if docker exec "$CONTAINER_NAME" systemctl is-system-running 2>/dev/null | grep -qE "running|degraded"; then
        SYSTEMD_READY=true
        echo "Systemd is ready (attempt $i)"
        break
    fi

    if [ $i -eq 60 ]; then
        echo "ERROR: Timeout waiting for systemd after 60 seconds"
        echo "=== Container logs ==="
        docker logs "$CONTAINER_NAME" 2>&1 || true
        echo "=== Systemd status ==="
        docker exec "$CONTAINER_NAME" systemctl status 2>&1 || true
        echo "=== Failed units ==="
        docker exec "$CONTAINER_NAME" systemctl --failed 2>&1 || true
        exit 1
    fi
    sleep 1
done

echo "Waiting for test environment initialization..."
for i in {1..30}; do
    STATUS=$(docker exec "$CONTAINER_NAME" systemctl is-active init-test-env.service 2>/dev/null || echo "unknown")
    if [ "$STATUS" = "active" ]; then
        echo "Test environment initialized"
        break
    fi
    if [ "$STATUS" = "failed" ]; then
        echo "ERROR: init-test-env.service failed"
        docker exec "$CONTAINER_NAME" journalctl -u init-test-env.service --no-pager 2>&1 || true
        exit 1
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Timeout waiting for init-test-env.service"
        docker exec "$CONTAINER_NAME" systemctl status init-test-env.service 2>&1 || true
        docker exec "$CONTAINER_NAME" journalctl -u init-test-env.service --no-pager 2>&1 || true
        exit 1
    fi
    sleep 1
done

echo "Starting integration tests..."
docker exec "$CONTAINER_NAME" systemctl start integration-tests.service &
START_PID=$!

# Wait a moment for service to start
sleep 3

# Follow the journal for test output
echo "=== Test output ==="
docker exec "$CONTAINER_NAME" journalctl -u integration-tests.service -f --no-pager 2>&1 &
JOURNAL_PID=$!

# Wait for service to complete
while true; do
    STATUS=$(docker exec "$CONTAINER_NAME" systemctl is-active integration-tests.service 2>/dev/null || echo "unknown")
    if [ "$STATUS" != "activating" ] && [ "$STATUS" != "active" ]; then
        break
    fi
    sleep 2
done

# Give journal time to flush
sleep 1
kill $JOURNAL_PID 2>/dev/null || true

# Get the exit code
EXIT_CODE=$(docker exec "$CONTAINER_NAME" systemctl show integration-tests.service --property=ExecMainStatus --value 2>/dev/null || echo "1")
echo ""
echo "=== Test service exit code: $EXIT_CODE ==="

if [ "$EXIT_CODE" = "0" ]; then
    echo "Integration tests PASSED"
else
    echo "Integration tests FAILED"
    # Show any additional failure info
    docker exec "$CONTAINER_NAME" systemctl status integration-tests.service 2>&1 || true
fi

exit "${EXIT_CODE:-1}"
