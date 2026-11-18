FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates git tzdata \
    && mkdir /app/skillhub-data \
    && chown nobody:nobody /app /app/skillhub-data

ARG TARGETOS
ARG TARGETARCH

COPY ${TARGETOS}/${TARGETARCH}/skillhub-server /app/bin/main

USER nobody

ENTRYPOINT ["/app/bin/main"]
