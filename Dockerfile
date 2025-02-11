FROM golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /ladon .

FROM alpine:3 AS run

WORKDIR /

COPY --from=build /ladon /ladon

EXPOSE 4000
ENTRYPOINT ["/ladon"]
