# IoT with MQTT and Kubernetes

This repository contains:

- An example IoT thing: the [fridge](fridge/).
- A Kubernetes operator to instantiate cloud controllers for things.

Run `go get` at the root of the repo.

We will need one MQTT hub with its `ca.pem` and two devices with their `crt.pem` and `key.pem`.

## Fridge

Go to the `fridge/` folder and run `go run main.go --help`.

```
Usage:
  fridge CLIENT_ID [TOPIC=/fr/fridge/51966]

Flags:
  -c, --client-crt string   Path to client certificate (default "crt.pem")
  -k, --client-key string   Path to client key (default "key.pem")
      --help                Print help
  -s, --server-crt string   Path to server certificate (default "ca.pem")
  -u, --url string          Server URL to connect to (default "tls://iot.fr-par.scw.cloud:8883")
```

1. Add a device to your hub and put its certificate and key into `crt.pem` and `key.pem`.
2. Put the hub certificate into `ca.pem`.
3. Run `go run main.go <YOUR_CLIENT_ID>`.

### State

The fridge should now be up and running and broadcasting its state to the MQTT hub every second.

```
2020/09/14 11:48:03 {"T":25,"D":4,"O":false}
2020/09/14 11:48:04 {"T":24,"D":4,"O":false}
2020/09/14 11:48:05 {"T":23,"D":4,"O":false}
```

| Abbreviation | Name                | Description                               |
| ------------ | ------------------- | ----------------------------------------- |
| T            | Temperature         | The inner temperature of the fridge       |
| D            | Desired temperature | The desired temperature inside our fridge |
| O            | Open                | Is the door open                          |

### Commands

The fridge can be operated through your keyboard:

| Key        | Action                                                    |
| ---------- | --------------------------------------------------------- |
| O          | Open the door                                             |
| C          | Close the door                                            |
| Arrow Up   | Increase the desired temperature by 1                     |
| Arrow Down | Decrease the desired temperature by 1                     |
| I          | Print fridge state                                        |
| A          | Emulate an alert by sending a message on `${topic}/alert` |

## Kubernetes Operator

You must have a Kubernetes cluster to install the operator. The easiest way to run a Kubernetes cluster locally in a docker container is [k3d](https://github.com/rancher/k3d).

Go to the [operator/](operator/) folder and run `make install` to install custom resource definitions managed by the operator to your cluster.

### Available resources

| Kind   | API Version       | Description                             |
| ------ | ----------------- | --------------------------------------- |
| Fridge | iot.ctison.dev/v1 | Monitor a fridge thing through MQTT hub |

### Fridge spec

| Field | Description                                                         |
| ----- | ------------------------------------------------------------------- |
| topic | Topic to listen for and where to publish alerts (`${topic}/alerts`) |

### Example configuration

Let's create the following resource in our cluster by running `kubectl apply -f config/samples/iot_v1_fridge.yaml`.

```yaml
apiVersion: iot.ctison.dev/v1
kind: Fridge
metadata:
  name: fridge-sample
spec:
  topic: /fr/fridge/51966
```

### Run the operator

The operator has the following usage:

```
Usage: fridge-operator CLIENT_ID

Available flags:
  -client-cert string
        Path to client certificate (default "crt.pem")
  -client-key string
        Path to client key (default "key.pem")
  -enable-leader-election
        Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
  -master --kubeconfig
        (Deprecated: switch to --kubeconfig) The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
  -metrics-addr string
        The address the metric endpoint binds to. (default ":8080")
  -server-cert string
        Path to server certificate (default "ca.pem")
  -url string
        URL of the mqtt server. (default "tls://iot.fr-par.scw.cloud:8883")
```

1. Add a device to your hub and put its certificate and key into `crt.pem` and `key.pem`.
2. Put the hub certificate into `ca.pem`.
3. Run `go run main.go <YOUR_CLIENT_ID>`.

The operator should now spawn a controller for our `Fridge` resource and this controller should start dumping received messages from the hub in the configured topic. Every time the controller detects the door of the fridge is open, it should send an alert to `${topic}/alert` asking to close the door!
