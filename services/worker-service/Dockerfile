FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o worker-service ./cmd/app/main.go

FROM nvidia/cuda:12.2.0-runtime-ubuntu22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    xz-utils \
    libx11-6 \
    libxi6 \
    libxrender1 \
    libxxf86vm1 \
    libfontconfig1 \
    libxcursor1 \
    libxinerama1 \
    libgl1 \
    libegl1 \
    libxfixes3 \
    libxkbcommon0 \
    libglu1-mesa \
    libsm6 \
    libice6 \
    libxext6 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN curl -L https://download.blender.org/release/Blender4.2/blender-4.2.0-linux-x64.tar.xz -o blender.tar.xz \
    && mkdir -p /opt/blender \
    && tar -xvf blender.tar.xz -C /opt/blender --strip-components=1 \
    && rm blender.tar.xz
ENV BLENDER_PATH=/opt/blender/blender
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility,graphics

WORKDIR /app

COPY --from=builder /app/worker-service .
COPY config/config.yaml ./config/config.yaml

EXPOSE 8081

CMD ["./worker-service"]
