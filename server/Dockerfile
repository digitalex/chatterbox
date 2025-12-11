# Stage 1: Build the binary
FROM golang:1.24 as builder
WORKDIR /app

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
# Build a static binary named 'server'
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server

# Stage 2: Create the runtime image (tiny)
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/server /server

# Cloud Run expects us to listen on port 8080
ENV PORT=8080
CMD ["/server"]

