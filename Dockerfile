FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apx ./cmd/apx

FROM alpine:latest

# Install necessary tools
RUN apk --no-cache add \
    ca-certificates \
    git \
    protobuf \
    nodejs \
    npm

# Install additional tools
RUN npm install -g @stoplight/spectral-cli

# Install Go-based tools
RUN apk add --no-cache go && \
    go install github.com/bufbuild/buf/cmd/buf@latest && \
    go install github.com/oasdiff/oasdiff@latest && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    apk del go

# Copy the binary
COPY --from=builder /app/apx /usr/local/bin/apx

# Copy configuration examples
COPY apx.example.yaml /etc/apx/apx.example.yaml

WORKDIR /workspace

ENTRYPOINT ["apx"]