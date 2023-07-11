# Manual Testing

## Getting Started

All commands are run from the root of the project, not this directory.

### Setup a local cluster

```sh
k3d cluster create barebones \
    --api-port 6550 \
    --servers 1 \
    --port 80:80@loadbalancer \
    --port 443:433@loadbalancer \
    --wait
```

### Install the operator CRDS

```sh
make install
```

### Run Operator

```sh
# or run the Go application in your favourite IDE etc.
make run
``` 

## Setting up the NATS, the Operator and System Account

### Create the Operator and System Account

```sh
kubectl apply -f examples/operator.yaml
```

The `User` is not mandatory, but you can download the credentials and can be useful for manually verifying things with 
the local `nats` command line tool.

### Download the configuration from cluster

Anything under `examples/nats-config` is git-ignored, so you can download the configuration from the cluster and run a 
local NATS server without worrying about wrong config etc.

Download the operator JWT:

```sh
kubectl get secret -n default operator-jwt -o json \
    | jq -r '.data["nats.jwt"]' \
    | base64 -d > examples/nats-config/operator/jwt
```

Find the system account public key and use it to replace the placeholder in the config:

```sh
SYSTEM_ACCOUNT=$(kubectl get account system -o jsonpath='{.status.keyPair.publicKey}')
sed "s/%SYSTEM_ACCOUNT%/$SYSTEM_ACCOUNT/" examples/nats.example.conf > examples/nats-config/nats.conf
```

### Run NATS

```sh
docker run --rm --name nao-nats -d -p 4222:4222 -v ${PWD}/examples/nats-config:/etc/nats-config nats --js --config /etc/nats-config/nats.conf
```

## Verify the things

There are two files, `account.yaml` and `account2.yaml` within the `examples` directory. You can download the 
credentials like so:

```sh
kubectl get secret -n default test-user-nats-creds -o json | jq -r '.data["nats.creds"]' | base64 -d > nats.creds
```

Then interact with NATS like so...

Get account info from our newly minted test account:

> If you run into an issue, check your `nats context` isn't set up to another cluster with TLS enabled etc.

```sh
nats -s nats://localhost:4222 --creds nats.creds account info
```

Download the system user credentials to a nats-sys.creds file, and you can view the accounts the NATS server is aware 
of:

```sh
nats -s nats://localhost:4222 --creds nats-sys.creds req '$SYS.REQ.CLAIMS.LIST' '' | jq
```

You can then proceed to create/delete/edit things and verify things are updated.

## Cleanup

Figure it out :)
