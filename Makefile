BINARY_NAME=containeruntime
GOOS=linux
CONTAINER_ENGINE?=podman

.PHONY: fmt vet lint test test-shell test-smoke test-stress test-runtime vuln tidy build build-all all setup-ubuntu setup-stress

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

test:
	go test -v -race -coverprofile=coverage.out ./...

test-shell: build setup-ubuntu
	./scripts/test_shell.sh

test-smoke: build setup-ubuntu
	./scripts/test_smoke.sh

test-stress: build setup-stress
	./scripts/test_stress.sh

test-runtime: test-smoke test-stress

vuln:
	govulncheck ./...

tidy:
	go mod tidy
	go mod verify

build:
	GOOS=$(GOOS) go build -o $(BINARY_NAME) cmd/main.go

build-all:
	go build ./...

all: fmt vet lint test-runtime vuln build build-all

setup-ubuntu:
	$(CONTAINER_ENGINE) create --name temp-ubuntu ubuntu:22.04
	mkdir -p /root/testbundle/ubuntufs
	$(CONTAINER_ENGINE) export temp-ubuntu -o /tmp/ubuntu.tar
	tar -xf /tmp/ubuntu.tar -C /root/testbundle/ubuntufs
	rm /tmp/ubuntu.tar
	$(CONTAINER_ENGINE) rm temp-ubuntu
	printf '%s\n' '{' \
	'"ociVersion": "1.0.2",' \
	'"process": {' \
		'"terminal": true,' \
		'"user": { "uid": 0, "gid": 0 },' \
		'"args": ["/bin/bash"],' \
		'"env": [' \
			'"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",' \
			'"TERM=xterm"' \
		'],' \
		'"cwd": "/",' \
		'"capabilities": {' \
			'"bounding": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"effective": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"inheritable": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"permitted": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"ambient": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"]' \
		'},' \
		'"rlimits": [{ "type": "RLIMIT_NOFILE", "hard": 1024, "soft": 1024 }],' \
		'"noNewPrivileges": true' \
	'},' \
	'"root": { "path": "/root/testbundle/ubuntufs", "readonly": false },' \
	'"hostname": "oci-ubuntu",' \
	'"mounts": [' \
		'{ "destination": "/proc", "type": "proc", "source": "proc" },' \
		'{ "destination": "/dev", "type": "tmpfs", "source": "tmpfs", "options": ["nosuid","strictatime","mode=755","size=65536k"] },' \
		'{ "destination": "/dev/pts", "type": "devpts", "source": "devpts", "options": ["nosuid","noexec","newinstance","ptmxmode=0666","mode=0620","gid=5"] },' \
		'{ "destination": "/dev/shm", "type": "tmpfs", "source": "shm", "options": ["nosuid","noexec","nodev","mode=1777","size=65536k"] },' \
		'{ "destination": "/dev/mqueue", "type": "mqueue", "source": "mqueue", "options": ["nosuid","noexec","nodev"] },' \
		'{ "destination": "/sys", "type": "sysfs", "source": "sysfs", "options": ["nosuid","noexec","nodev","ro"] }' \
	'],' \
	'"linux": {' \
		'"resources": { "devices": [{ "allow": false, "access": "rwm" }] },' \
		'"namespaces": [' \
			'{ "type": "pid" },' \
			'{ "type": "network" },' \
			'{ "type": "ipc" },' \
			'{ "type": "uts" },' \
			'{ "type": "mount" }' \
		'],' \
		'"maskedPaths": [' \
			'"/proc/kcore", "/proc/latency_stats", "/proc/timer_list", "/proc/timer_stats",' \
			'"/proc/sched_debug", "/proc/scsi", "/sys/firmware"' \
		'],' \
		'"readonlyPaths": [' \
			'"/proc/asound", "/proc/bus", "/proc/fs", "/proc/irq",' \
			'"/proc/sys", "/proc/sysrq-trigger"' \
		']' \
	'}' \
	'}' > /root/testbundle/config.json

setup-stress:
	$(CONTAINER_ENGINE) create --name temp-stress progrium/stress
	mkdir -p /root/testbundle/stressfs
	$(CONTAINER_ENGINE) export temp-stress -o /tmp/stress.tar
	tar -xf /tmp/stress.tar -C /root/testbundle/stressfs
	rm /tmp/stress.tar
	$(CONTAINER_ENGINE) rm temp-stress
	printf '%s\n' '{' \
	'"ociVersion": "1.0.2",' \
	'"process": {' \
		'"terminal": false,' \
		'"user": { "uid": 0, "gid": 0 },' \
		'"args": ["/usr/bin/stress -c 1"],' \
		'"env": [' \
			'"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",' \
			'"TERM=xterm"' \
		'],' \
		'"cwd": "/",' \
		'"capabilities": {' \
			'"bounding": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"effective": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"inheritable": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"permitted": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"],' \
			'"ambient": ["CAP_AUDIT_WRITE","CAP_KILL","CAP_NET_BIND_SERVICE"]' \
		'},' \
		'"rlimits": [{ "type": "RLIMIT_NOFILE", "hard": 1024, "soft": 1024 }],' \
		'"noNewPrivileges": true' \
	'},' \
	'"root": { "path": "/root/testbundle/stressfs", "readonly": false },' \
	'"hostname": "oci-stress",' \
	'"mounts": [' \
		'{ "destination": "/proc", "type": "proc", "source": "proc" },' \
		'{ "destination": "/dev", "type": "tmpfs", "source": "tmpfs", "options": ["nosuid","strictatime","mode=755","size=65536k"] },' \
		'{ "destination": "/dev/pts", "type": "devpts", "source": "devpts", "options": ["nosuid","noexec","newinstance","ptmxmode=0666","mode=0620","gid=5"] },' \
		'{ "destination": "/dev/shm", "type": "tmpfs", "source": "shm", "options": ["nosuid","noexec","nodev","mode=1777","size=65536k"] },' \
		'{ "destination": "/dev/mqueue", "type": "mqueue", "source": "mqueue", "options": ["nosuid","noexec","nodev"] },' \
		'{ "destination": "/sys", "type": "sysfs", "source": "sysfs", "options": ["nosuid","noexec","nodev","ro"] }' \
	'],' \
	'"linux": {' \
		'"resources": { "devices": [{ "allow": false, "access": "rwm" }] },' \
		'"namespaces": [' \
			'{ "type": "pid" },' \
			'{ "type": "network" },' \
			'{ "type": "ipc" },' \
			'{ "type": "uts" },' \
			'{ "type": "mount" }' \
		'],' \
		'"maskedPaths": [' \
			'"/proc/kcore", "/proc/latency_stats", "/proc/timer_list", "/proc/timer_stats",' \
			'"/proc/sched_debug", "/proc/scsi", "/sys/firmware"' \
		'],' \
		'"readonlyPaths": [' \
			'"/proc/asound", "/proc/bus", "/proc/fs", "/proc/irq",' \
			'"/proc/sys", "/proc/sysrq-trigger"' \
		']' \
	'}' \
	'}' > /root/testbundle/config-stress.json

lint:
	golangci-lint run
