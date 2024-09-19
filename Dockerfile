FROM golang:alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Install basic packages
RUN apk add \
    gcc \
    g++

# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY . .

# Download all the dependencies
RUN go mod download

# Build image
RUN go build .

FROM alpine:latest AS runner

WORKDIR /app

COPY --from=builder /app/rsshub-smart-layer /app/app

# Run the executable
CMD ["/app/app"]
