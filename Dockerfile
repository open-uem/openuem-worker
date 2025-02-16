FROM golang:1.23.6 AS build
COPY . ./
RUN go build -o /bin/openuem-worker .

FROM debian:latest
RUN apt-get update && apt install -y ca-certificates
COPY --from=build /bin/openuem-worker /bin/openuem-worker
WORKDIR /tmp
ENTRYPOINT ["/bin/openuem-worker"]