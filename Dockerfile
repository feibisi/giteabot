FROM golang:1.26-alpine AS build

WORKDIR /src
COPY . .
RUN go build -o /out/giteabot .

FROM alpine:3.19
COPY --from=build /out/giteabot /usr/local/bin/giteabot
ENTRYPOINT ["/usr/local/bin/giteabot"]
