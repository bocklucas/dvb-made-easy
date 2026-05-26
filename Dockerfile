# Stage 1: Build the frontend (SvelteKit SPA)
FROM node:22-alpine AS frontend-builder
WORKDIR /web

COPY web/package.json web/package-lock.json* ./
RUN npm install

COPY web/ .
RUN npm run build

# Stage 2: Build the backend (Go Server embedding the frontend build)
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN go mod tidy

# Copy frontend build directory from Stage 1 to the Go embed target directory
COPY --from=frontend-builder /web/build /app/internal/api/build

# Build statically linked binary with CGO disabled
RUN CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -o /server ./cmd/server

# Stage 3: Final minimal runtime container
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy built server binary from Stage 2
COPY --from=backend-builder /server /server

EXPOSE 7331

ENTRYPOINT ["/server"]
CMD ["--config-dir", "/app/config"]
