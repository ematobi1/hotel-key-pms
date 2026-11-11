FROM golang:1.26-alpine AS builder
ARG BUILD_VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${BUILD_VERSION}" \
    -o /bin/service .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/service /service
EXPOSE 8080
ENTRYPOINT ["/service"]
