This is a simple project to help bootstrapping a go webserver in the cloud

It contains a basic "hello world" webserver that can be run on a docker container

It is prepaared to have dev and prod setup with different env variables (defined in yaml file)


## Run

### Locally with go

Simply do the obvious `go run main.go`

### Locally with Docker

build `docker build -t hello-go .`
and run `docker run -p 8080:8080 --env-file env.local.env hello-go`

in both cases open `http://localhost:8080/hello` to check is running

## Deploy

### Setup (one time)
Create project: `gcloud projects create <project-name>`

Set project as default: `gcloud config set project <project-name>`

(Enable billing on the project if needed)

Enable APIs:

`gcloud services enable run.googleapis.com artifactregistry.googleapis.com cloudbuild.googleapis.com`

### Manual Deploy to GCP:

`gcloud run deploy <service-name> --source . --region us-central1  --allow-unauthenticated --env-vars-file <env.xxx.yaml>`

### Github actions

#### Setup

get gcp project number

 `PROJECT_NUMBER=$(gcloud projects describe $(gcloud config get-value project) --format="value(projectNumber)")`


create pool

`gcloud iam workload-identity-pools create "github-pool" --location="global" --display-name="GitHub Pool"`

create service account 

`gcloud iam service-accounts create github-actions-sa \
    --display-name="GitHub Actions Service Account"`

handshake:

`gcloud iam workload-identity-pools providers create-oidc "github-provider" \
  --location="global" \
  --workload-identity-pool="github-pool" \
  --display-name="GitHub Provider" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
  --attribute-condition="assertion.repository_owner == '<GITHUB_USER>'"`

  add permissions - ask ai to add needed ones

  

