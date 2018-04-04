FROM progrium/busybox
MAINTAINER Guangzheng Zhang <zhang.elinks@gmail.com>
WORKDIR /
COPY openshift-api-proxy /
ENTRYPOINT ["/openshift-api-proxy"]
