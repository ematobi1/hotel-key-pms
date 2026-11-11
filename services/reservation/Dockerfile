FROM golang:1.27.0-alpine AS builder
ARG BUILD_VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${BUILD_VERSION}" \
    -o /bin/reservation .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/reservation /reservation
EXPOSE 8080
ENTRYPOINT ["/reservation"]
