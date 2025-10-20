# Builder container
FROM --platform=linux/amd64 registry.access.redhat.com/ubi9/go-toolset AS builder

ARG GCFLAGS=""

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

USER 0
RUN make build

# Final runtime image
FROM --platform=linux/amd64 registry.access.redhat.com/ubi9/ubi-minimal

WORKDIR /app

COPY --from=builder /app/bin/dcm-placement-api /app/

# Use non-root user
RUN chown -R 1001:0 /app
USER 1001

# Run the server
EXPOSE 8080
ENTRYPOINT ["/bin/bash", "-c", "/app/dcm-placement-api run"]
