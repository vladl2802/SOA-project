ARG GRPC_PORT

FROM soa-project-shared:latest as shared

FROM golang:1.23-alpine as build

# RUN apk add --no-cache make
# RUN apk add --no-cache protoc-gen-go

WORKDIR /app/post-service/src
COPY --from=shared app/shared /app/shared
COPY post-service/src /app/post-service/src/

# RUN make compile_proto
RUN go mod download

RUN --mount=type=bind,target=~/.cache/go-build go build -o /app/bin/post-service

FROM alpine:3.14 as run

WORKDIR /app/bin
COPY --from=build app/bin/post-service /app/bin/post-service
# Temporal solution. Proper will be to split this Dockerfile into two: proto builder and service build
COPY --from=build app/post-service/src /app/post-service/src

EXPOSE $GRPC_PORT

ENTRYPOINT ["/app/bin/post-service"]

