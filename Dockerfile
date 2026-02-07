# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/ai-critic-react
COPY ai-critic-react/package.json ai-critic-react/package-lock.json ./
RUN npm install
COPY ai-critic-react/ ./
RUN npm run build

# Stage 2: Build Go server
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/ai-critic-react/dist ai-critic-react/dist
RUN CGO_ENABLED=0 go build -ldflags="" -o /ai-critic ./

# Stage 3: Minimal runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends git curl ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=go-builder /ai-critic /usr/local/bin/ai-critic

EXPOSE 23712
ENTRYPOINT ["ai-critic"]
CMD ["--port", "23712"]
