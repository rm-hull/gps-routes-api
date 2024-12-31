FROM golang:1.23 AS build
WORKDIR /go/src
COPY . .

ENV CGO_ENABLED=0

RUN go build -o gps-routes-server .

FROM scratch AS runtime
ENV GIN_MODE=release
COPY --from=build /go/src/gps-routes-server ./
EXPOSE 8080/tcp
ENTRYPOINT ["./gps-routes-server"]
