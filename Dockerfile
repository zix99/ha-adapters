FROM golang:1.18-alpine AS gobuild
WORKDIR /opt/ha-adapters
COPY go.* ./
RUN go mod download
COPY . .

RUN go build ha-adapters/cmd/ad410

# Final iamge
FROM alpine:latest
WORKDIR /opt/ha-adapters
COPY --from=gobuild /opt/ha-adapters/ad410 .

ENTRYPOINT ["./ad410"]