FROM alpine
RUN apk add --no-cache bash ca-certificates tzdata
COPY web-indexer /usr/bin/web-indexer
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /usr/bin/web-indexer /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
