FROM golang as builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN go build -o server

FROM debian
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
RUN apt-get update && apt-get install -y curl
RUN curl https://cursor.com/install -fsS | bash
RUN ln -s /root/.local/bin/cursor-agent /usr/bin/
COPY --from=builder /src/server /
CMD ["/server"]
