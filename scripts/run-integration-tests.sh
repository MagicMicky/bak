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
    "$IMAGE_NAME"

echo "Waiting for systemd to be ready..."
for i in {1..30}; do
    if docker exec "$CONTAINER_NAME" systemctl is-system-running --wait 2>/dev/null; then
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Timeout waiting for systemd"
        docker logs "$CONTAINER_NAME"
        exit 1
    fi
    sleep 1
done

echo "Waiting for test environment initialization..."
for i in {1..30}; do
    STATUS=$(docker exec "$CONTAINER_NAME" systemctl is-active init-test-env.service 2>/dev/null || echo "unknown")
    if [ "$STATUS" = "active" ]; then
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Timeout waiting for init-test-env.service"
        docker exec "$CONTAINER_NAME" journalctl -u init-test-env.service --no-pager || true
        exit 1
    fi
    sleep 1
done

echo "Starting integration tests..."
docker exec "$CONTAINER_NAME" systemctl start integration-tests.service &

# Follow the journal for test output
sleep 2
docker exec "$CONTAINER_NAME" journalctl -u integration-tests.service -f --no-pager &
JOURNAL_PID=$!

# Wait for service to complete
while docker exec "$CONTAINER_NAME" systemctl is-active integration-tests.service 2>/dev/null | grep -q "^activating\|^active"; do
    sleep 2
done

kill $JOURNAL_PID 2>/dev/null || true

# Get the exit code
EXIT_CODE=$(docker exec "$CONTAINER_NAME" systemctl show integration-tests.service --property=ExecMainStatus --value 2>/dev/null || echo "1")
echo ""
echo "Test service exit code: $EXIT_CODE"

if [ "$EXIT_CODE" = "0" ]; then
    echo "Integration tests PASSED"
else
    echo "Integration tests FAILED (exit code: $EXIT_CODE)"
fi

exit "${EXIT_CODE:-1}"
