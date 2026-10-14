FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/portal-api . \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/project-service ./cmd/project-service

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates && addgroup -g 10001 app && adduser -D -u 10001 -G app app
WORKDIR /app
USER 10001:10001
ENV GIN_MODE=release

FROM runtime AS portal-api
COPY --from=build /out/portal-api /app/portal-api
ENV PORTAL_LISTEN_ADDR=:8080 FRONTEND_DIR="" PROJECT_SERVICE_URL=http://project-service:8082
EXPOSE 8080
ENTRYPOINT ["/app/portal-api"]

FROM runtime AS project-service
COPY --from=build /out/project-service /app/project-service
ENV PROJECT_LISTEN_ADDR=:8082 PROJECT_STORE=dynamodb
EXPOSE 8082
ENTRYPOINT ["/app/project-service"]
