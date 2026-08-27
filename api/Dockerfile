# ---- build ----
FROM golang@sha256:ae5a2316d12f3e78fd99177dad452e6ad4f240af2d71d57b480c3477f250fec6 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

# ---- runtime ----
FROM gcr.io/distroless/static@sha256:f2ea2709ac8db56323cbd7d014277f32cb572d9ea124b0076f7aafe5980678fe
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]