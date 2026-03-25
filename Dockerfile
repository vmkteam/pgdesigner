FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY pgdesigner /usr/local/bin/pgdesigner
WORKDIR /work
EXPOSE 8080
ENTRYPOINT ["pgdesigner"]
