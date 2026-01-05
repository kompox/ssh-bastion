# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/ssh-bastion ./cmd/ssh-bastion

FROM alpine:3.20

RUN apk add --no-cache ca-certificates openssh-server

# Runtime files
WORKDIR /app
COPY web/ /app/web/
COPY --from=build /out/ssh-bastion /usr/local/bin/ssh-bastion
COPY --chmod=755 docker/ssh-entrypoint.sh /usr/local/bin/ssh-entrypoint
COPY --chmod=755 docker/ssh-bastion-shell /usr/local/bin/ssh-bastion-shell

# Create the shared SSH user at build time so we don't need to edit /etc/passwd
# at runtime. The shell is a wrapper that always runs the forcecommand script.
RUN adduser -D -h /home/jump -s /usr/local/bin/ssh-bastion-shell jump

# Ensure the account is not locked so sshd accepts publickey auth.
RUN passwd -u jump >/dev/null 2>&1 || true \
	&& passwd -d jump >/dev/null 2>&1 || true

# Useful defaults for local testing
ENV SSHBASTION_DATA_DIR=/data

EXPOSE 22 8080

ENTRYPOINT ["ssh-bastion"]
CMD ["serve", "-http-addr", ":8080"]
