# ---- build ----
FROM golang@sha256:079e59808d2d252516e27e3f3a9c003740dee7f75e55aa71528766d52bcfc16a AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

# ---- runtime ----
FROM gcr.io/distroless/static@sha256:d5f030ca7c5793784e9ea4178a116da360250411d13921a5af27c6cb5a5949bf
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]