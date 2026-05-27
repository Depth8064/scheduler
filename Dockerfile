# Build the scheduler binary using Go 1.26
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /scheduler ./cmd/scheduler

FROM gcr.io/distroless/static-debian11
COPY --from=builder /scheduler /scheduler
EXPOSE 8080
ENTRYPOINT ["/scheduler"]
