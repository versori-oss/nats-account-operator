# nats-account-operator

The NATS Account Operator provides a declarative approach to managing Authentication and Authorization infrastructure
using the [NATS Decentralized JWT][nats-authnz] mechanisms.

## Description

There are four CRD types implemented by the operator:

- `Operator` - Represents a NATS Operator, the administrative entity of a NATS cluster.
- `Account` - Represents a NATS Account to be managed by an Operator.
- `User` - Represents a NATS User which exists within an Account.
- `SigningKey` - Represents a public/private key pair used to sign JWTs.

Further details of the CRD types can be found in the [Specification](./docs/specification.md) documentation.

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

    ```sh
    kubectl apply -f config/samples/
    ```

2. Build and push your image to the location specified by `IMG`:

    ```sh
    make docker-build docker-push IMG=<some-registry>/nats-accounts-operator:tag
    ```

3. Deploy the controller to the cluster with the image specified by `IMG`:

    ```sh
    make deploy IMG=<some-registry>/nats-accounts-operator:tag
    ```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing

View the [Development Guide](./docs/development-guide.md) for info on running locally and contributing bug fixes/new
features.

### How it works

This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the
cluster

### Test It Out
1. Install the CRDs into the cluster:

    ```sh
    make install
    ```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

    ```sh
    make run
    ```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

See [LICENSE][license]

[nats-authnz]: https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt
[license]: ./LICENSE
