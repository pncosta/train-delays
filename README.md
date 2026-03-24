This is a WIP project to track delays and ponctuality of the portuguese trains.





## Run

### Locally with go

Simply do the obvious `go run .`

### Locally with Docker

build `docker build -t hello-go .`
and run `docker run -p 8080:8080 --env-file env.local.env hello-go`

in both cases open `http://localhost:8080/hello` to check is running

## Deploy

### gcloud run

gcloud run deploy backend-dev \
  --source . \
  --region=europe-southwest1 \
  --env-vars-file env.dev.yaml




### Github actions
deployment to dev is automatic on push
deployment to prod is manually via gh actions

### DB

data is being scrapped daily and stored on a SQLite DB kept in a gcp  bucket

in order to setup on GCP:

1  - create bucket:

```
gcloud storage buckets create gs://pt-train-delays-db --location=europe-southwest1
```

2 - mount bucket to service:
```
gcloud run services update backend-dev \
  --region=europe-southwest1 \
  --add-volume=name=db-volume,type=cloud-storage,bucket=pt-train-delays-db-dev \
  --add-volume-mount=volume=db-volume,mount-path=/data
  ```



