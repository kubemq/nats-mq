# nats-mq

Simple bridge between NATS streaming and MQ Series

## Notes/Caveats

* This bridge depends on `github.com/ibm-messaging/mq-golang` which uses CGO to access the MQI libraries. 
* Request/reply with queues is supported but reply-to topics are not.
* Request/reply with NATS streaming requires that ExcludeHeaders be configured to False, the reply-to channel is in the header. Clients need
to use the BridgeMessage class to wrap messages on the streaming side.
* For testing we embed the nats-streaming-server which brings in a fair number of dependencies. The bridge executable only requires the nats and streaming clients as well as the go mq-series library.

## Developing

### The MQSeries library

The go [mq series library](https://github.com/ibm-messaging/mq-golang) requires the client libraries. These are referenced from the readme, except for [MacOS which are available here](https://developer.ibm.com/messaging/2019/02/05/ibm-mq-macos-toolkit-for-developers/).

```bash
export MQ_INSTALLATION_PATH=<your installation library>
export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
export CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"
export CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
 ```

 *Note there is a typo on the ibm mq web page, missing `-rpath` and has `rpath` instead.

 Build the MQ library:

 ```bash
 go install ./ibmmq
 go install ./mqmetric
 ```

 you may see `ld: warning: directory not found for option '-L/opt/mqm/lib64'` but you can ignore it.

 You will also need to set these variables for building the bridge itself, since it depends on the MQ series packages.

 The dependency on the MQ package requires v3.3.4 to fix an rpath issue on Darwin.

#### Running the examples from the Go library

The examples that pub/sub require an environment that tells them where the server is. You can use something like:

```bash
% export MQSERVER="DEV.APP.SVRCONN/TCP/localhost(1414)"
```

for the default docker setup described below. This will allow you to run examples:

```bash
% go run amqsput.go DEV.QUEUE.1 QM1
```

### Running the docker container for MQ Series

See [https://hub.docker.com/r/ibmcom/mq/](https://hub.docker.com/r/ibmcom/mq/) to get the docker container.

Also check out the [usage documentation](https://github.com/ibm-messaging/mq-container/blob/master/docs/usage.md).

```bash
docker run \
  --env LICENSE=accept \
  --env MQ_QMGR_NAME=QM1 \
  --publish 1414:1414 \
  --publish 9443:9443 \
  --detach \
  ibmcom/mq
```

or use the `scripts/run_mq.sh` to execute that command.

#### Connecting to the docker web admin

[https://localhost:9443/ibmmq/console/](https://localhost:9443/ibmmq/console/)

The login is:

```bash
User: admin
Password: passw0rd
```

Chrome may complain because the certificate for the server isn't valid, but go on through.

#### Connecting with an application

For applications the login is documented as:

```bash
User: app
Password: *none*
```

I found that simply turning off user name and password works.

#### TLS Setup

See [this ibm post](https://developer.ibm.com/messaging/learn-mq/mq-tutorials/secure-mq-tls/) for information about how the test TLS files for the docker image were created. The generated certs/keys are in the resources folder under mqm. A script is provided `scripts/run_mq_tls.sh` to run the server with this TLS setting. The TLS script will also set the app password to `passw0rd` for testing.

The server cert has the password `k3ypassw0rd`.

The client cert has the password `tru5tpassw0rd`, and the label `QM1.cert`

I created the kdb file using `runmqakm -cert -export -db client_key.p12 -pw tru5tpassw0rd -target_stashed -target_type kdb -target client.kdb -label "QM1.cert"`.