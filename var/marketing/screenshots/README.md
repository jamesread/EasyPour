# Marketing screenshots

The PNGs here are the screenshots used in the project README. They are
captured from a throwaway EasyPour instance that uses a temporary SQLite
database and fictional data, so every run gives the same result.

## Requirements

`repo-helper`, `sql-migrate`, Chrome/Chromium with a matching `chromedriver`,
Go, Node/npm, `curl` and `jq`.

## Regenerate

```bash
make            # from this directory, or `make screenshots` from the repo root
```

`make` builds the service and frontend, starts the backend on
`127.0.0.1:18480` with `service.yaml`, seeds it from `seed/` through the public
API, captures every section of `screenshots.ini`, regenerates
`screenshot-manifest.yaml`, then stops the backend. Everything temporary lives
in `tmp/` (gitignored).

To browse the seeded instance yourself, run `make build up seed`, open
http://127.0.0.1:18480/ and log in as `alex` / `demo-password`. Run
`make down` when you have finished.

## Adding or changing a screenshot

Edit `screenshots.ini`; each section needs `title` and `description`. Then
run `make`. `screenshot-manifest.yaml` is generated, so don't edit it by hand.
