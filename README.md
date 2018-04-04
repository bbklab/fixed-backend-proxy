### Description
Visit HTTP/HTTPS Services with prefix `/api/dmos/`

### Example
```bash
# docker run \
	--net=host \
	--name openshift-api-proxy  \
	-e LISTEN=:8080 \
	-e BACKEND_HTTPS=true \
	-e BACKEND_ENDPOINT=some.host:8443 \
	bbklab/openshift-api-proxy:latest
```
