FROM alpine:3.11
LABEL maintainer="support@kubermatic.com"

RUN apk add --no-cache ca-certificates

COPY aquayman /usr/local/bin/

USER nobody
ENTRYPOINT ["aquayman"]
