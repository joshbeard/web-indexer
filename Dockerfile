FROM alpine

RUN apk add --no-cache ca-certificates tzdata

COPY web-indexer /usr/local/bin/web-indexer

ENTRYPOINT ["web-indexer"]
