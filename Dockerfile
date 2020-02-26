FROM docker.io/golang:1.14 as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make test
RUN make build

FROM gcr.io/distroless/static:nonroot

COPY --from=build /app/lieutenant-operator /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/lieutenant-operator" ]
