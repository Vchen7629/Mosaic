FROM ubuntu:24.04 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    ca-certificates \
    libdlib-dev \
    libblas-dev \
    liblapack-dev \
    libopenblas-dev \
    pkg-config \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

RUN wget -q https://go.dev/dl/go1.24.1.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz \
    && rm go1.24.1.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o face-detection ./cmd/...


FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    libdlib19.1 \
    libopenblas0 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/face-detection .
COPY internal/models ./models

CMD ["./face-detection"]
