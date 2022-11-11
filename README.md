## Project

The aim of the project is to implement in [Go](https://go.dev/) two distributed election algorithms (_Chang and Roberts algorithm_ and _Bully algorithm_) and the following services:

- Register service.
- Heartbeat monitoring service.

>To deploy the application should be used _Docker_ containers on _EC2_ instance.
> 
## Execution

For the execution are required:

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

To install _Docker_ and _Docker Compose_ in one go, you can download [Docker Desktop](https://www.docker.com/products/docker-desktop/).

### Local Execution

The program can be run on _Linux_ and _Windows_.

The complete list of flags is as follows:

```bash
$ go run launch.go                                                    

usage: launch.go [-a {ring,bully}] [-n] [-hb] [-d] [-v] [-t {1,2,3}]

Implementation of distributed election algorithms \in Go.

optional arguments:
    -a {ring,bully}   election algoritm   
    -n                number of peers \in the network
    -hb               heartbeat service repeat \time
    -d                maximum random delay to forwarding messages
    -v                enable \verbosity 
    -t {1,2,3}        run one of the available tests
```

The _config.json_ file has been defined to manage the network settings (IP addresses, port numbers).

#### Tests

Test execution can be handled as:

```bash
go run launch.go -t {1,2,3} -n {\>=4}
```

The tests are as follows:

- Test 1: only one peer crashes, but it's not the leader.
- Test 2: only the leader crashes.
- Test 3: one peer and the leader crash.
