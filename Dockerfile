# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/ssh-bastion ./cmd/ssh-bastion

FROM alpine:3.20

RUN apk add --no-cache ca-certificates dnsmasq openssh-server

# Runtime files
WORKDIR /app
COPY web/ /app/web/
COPY --from=build /out/ssh-bastion /usr/local/bin/ssh-bastion
COPY --chmod=755 docker/ssh-entrypoint.sh /usr/local/bin/ssh-entrypoint

# Useful defaults for local testing
ENV SSHBASTION_DATA_DIR=/data

EXPOSE 22 8080

ENTRYPOINT ["ssh-bastion"]
CMD ["web", "-addr", ":8080"]
