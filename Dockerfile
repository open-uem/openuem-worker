FROM golang:1.24.4 AS build
COPY . ./
RUN go build -o /bin/openuem-worker .

FROM debian:latest
RUN apt-get update && apt install -y ca-certificates
COPY --from=build /bin/openuem-worker /bin/openuem-worker
WORKDIR /tmp
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD /bin/openuem-worker healthcheck || exit 1
ENTRYPOINT ["/bin/openuem-worker"]