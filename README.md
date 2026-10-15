# guestbook

A tiny Go web app to try [piway](https://github.com/OFK0/piway) with: sign
the book, read the latest entries, and watch a deploy swap the container
without downtime.

- `piway.yaml` tells piway how to build and run it
  ([manifest reference](https://github.com/OFK0/piway/blob/main/docs/MANIFEST.md)).
- `GET /healthz` is the health check piway waits for before switching
  traffic to a new version.
- Entries live in memory, so each deploy starts a fresh book.

Run it without piway:

```sh
go run .                      # http://localhost:3000
docker build -t guestbook . && docker run --rm -p 3000:3000 guestbook
```
