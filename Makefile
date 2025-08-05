BINARY_NAME=containeruntime
GOOS=linux

.PHONY: build setup-ubuntu lint

build:
	GOOS=$(GOOS) go build -o $(BINARY_NAME) .

setup-ubuntu:
	sudo docker create --name temp-ubuntu ubuntu:22.04
	sudo mkdir -p /root/ubuntufs
	sudo docker export temp-ubuntu -o /tmp/ubuntu.tar
	sudo tar -xf /tmp/ubuntu.tar -C /root/ubuntufs
	sudo rm /tmp/ubuntu.tar
	sudo docker rm temp-ubuntu

lint:
	go fmt ./...