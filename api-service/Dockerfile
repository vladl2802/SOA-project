FROM soa-project-shared:latest as shared

FROM soa-project-user-service:latest as user-service
FROM soa-project-post-service:latest as post-service

FROM golang:1.23-alpine as build

WORKDIR /app/api-service/src
COPY --from=shared app/shared /app/shared
COPY --from=user-service app/user-service/src /app/user-service/src
COPY --from=post-service app/post-service/src /app/post-service/src

COPY api-service/src /app/api-service/src

RUN go mod download

RUN --mount=type=bind,target=~/.cache/go-build go build -o /app/bin/api-service

FROM alpine:3.14 as run

WORKDIR /app/bin
COPY --from=build app/bin/api-service /app/bin/api-service
COPY config/signature.pub /config/signature.pub

ENV JWT_PUBLIC /config/signature.pub

EXPOSE 8080

ENTRYPOINT ["/app/bin/api-service"]

