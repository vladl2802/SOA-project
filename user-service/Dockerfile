ARG GRPC_PORT

FROM soa-project-shared:latest as shared

FROM golang:1.23-alpine as build

# RUN apk add --no-cache make
# RUN apk add --no-cache protoc-gen-go

WORKDIR /app/user-service/src
COPY --from=shared app/shared /app/shared
COPY user-service/src /app/user-service/src

# RUN make compile_proto
RUN go mod download

RUN --mount=type=bind,target=~/.cache/go-build go build -o /app/bin/user-service

FROM alpine:3.14 as run

WORKDIR /app/bin
COPY --from=build app/bin/user-service /app/bin/user-service
# Temporal solution. Proper will be to split this Dockerfile into two: proto builder and service build
COPY --from=build app/user-service/src /app/user-service/src

COPY config/signature.pem /config/signature.pem

ENV JWT_PRIVATE /config/signature.pem

EXPOSE $GRPC_PORT

ENTRYPOINT ["/app/bin/user-service"]

