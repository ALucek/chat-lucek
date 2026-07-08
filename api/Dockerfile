# ---- build ----
FROM golang@sha256:63f132d58c1f589f0dcda584933a9bb44bfda1150f1506377f5a902f34d86033 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

# ---- runtime ----
FROM gcr.io/distroless/static@sha256:3592aa8171c77482f62bbc4164e6a2d141c6122554ace66e5cc910cadb961ff0
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]