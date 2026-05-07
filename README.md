This is a WIP project that aims to track the ponctuality, delays and supressions of the portuguese trains from CP company.

It is build in go and is composed of:
- periodid job that scrapes the CP API to get the delays of each train travel and store it on a DB
- server to get the data
- web app to visualise the data 
-
## Run

### Env Vars

The following env vars are expected either on the scraper or web server:

```
CP_API_KEY: - API key from CP
CP_CLIENT_ID: - client ID from CP
CP_CLIENT_SECRET: client secret from CP 
TURSO_DB_URL: URL from sql DB
TURSO_DB_TOKEN: long live token to access the DB
```

### Locally with go

Simply do the obvious `go run .`

### Locally with Docker


build `docker build -t <name> .`
and run `docker run -p 8080:8080 --env-file env/env.local.env <name>`


## Deploy

### gcloud run

to deploy from machine directly to gcloud:

server:

gcloud run deploy backend-dev \
  --source . \
  --region=europe-southwest1 \
  --env-vars-file ../env/env.dev.yaml

job:
gcloud run jobs deploy scraper-job-dev \
  --source . \
  --region=europe-west1 \
  --env-vars-file ../env/env.dev.yaml


### Github actions
deployment to dev is automatic on push
deployment to prod is manually via gh actions

### DB

[Turso](https://turso.tech/) is being used as provider for a SQL DB



