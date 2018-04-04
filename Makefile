gitCommit=$(shell git rev-parse --short HEAD)

default: build

build: clean binary image

binary: 
	docker run --rm \
        --name buildproxy \
		-w /tmp \
        -e CGO_ENABLED=0 \
        -e GOOS=linux \
        -v $(PWD):/tmp \
        golang:1.9 \
        sh -c 'go build -o openshift-api-proxy main.go'

image:
	docker build --force-rm -t bbklab/openshift-api-proxy:$(gitCommit) -f Dockerfile .
	docker tag bbklab/openshift-api-proxy:$(gitCommit) bbklab/openshift-api-proxy:latest

clean:
	rm -fv openshift-api-proxy
