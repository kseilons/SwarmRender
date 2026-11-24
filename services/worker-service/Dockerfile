FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o worker-service ./cmd/app/main.go

FROM nvidia/cuda:13.2.1-cudnn-devel-ubuntu24.04

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
    libglvnd0 \
    libopengl0 \
    libglx0 \
    libgl1-mesa-dri \
    libglx-mesa0 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*


RUN curl -L https://download.blender.org/release/Blender5.0/blender-5.0.1-linux-x64.tar.xz -o blender.tar.xz \
    && mkdir -p /opt/blender \
    && tar -xvf blender.tar.xz -C /opt/blender --strip-components=1 \
    && rm blender.tar.xz
ENV BLENDER_PATH=/opt/blender/blender

WORKDIR /app

COPY --from=builder /app/worker-service .
COPY config/ config/

RUN groupadd -r appgroup && useradd -r -g appgroup -m -d /home/appuser appuser && \
    chown -R appuser:appgroup /app /opt/blender

USER appuser

EXPOSE 8081

CMD ["./worker-service"]
