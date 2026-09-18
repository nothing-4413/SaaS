FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/saas-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/saas-worker ./cmd/worker

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/saas-api /usr/local/bin/saas-api
COPY --from=build /out/saas-worker /usr/local/bin/saas-worker
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/saas-api"]
