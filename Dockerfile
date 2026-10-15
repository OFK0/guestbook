# Build a static binary, then ship it alone in an empty image.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go page.html ./
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /guestbook .

FROM scratch
COPY --from=build /guestbook /guestbook
USER 65534:65534
EXPOSE 3000
ENTRYPOINT ["/guestbook"]
