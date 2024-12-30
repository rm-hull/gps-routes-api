FROM golang:1.23 AS build
WORKDIR /go/src
COPY go ./go
COPY main.go .
COPY go.sum .
COPY go.mod .

ENV CGO_ENABLED=0

RUN go build -o gps-routes-api .

FROM scratch AS runtime
ENV GIN_MODE=release
COPY --from=build /go/src/gps-routes-api ./
EXPOSE 8080/tcp
ENTRYPOINT ["./gps-routes-api"]
