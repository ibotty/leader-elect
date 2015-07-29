# Master election using etcd to start and monitor systemd units

## Master election

`master-elect` will create a temporal lock in etcd, setting the hostname 
of the running master. The master will periodically reset the ttl while 
other nodes will periodically check whether the key is still there and
race to become master when that's not the case.

## Configuration

Configuration is done using the command line arguments and environment
variables. When using the provided systemd units, it will load 
configuraton from `/etc/master-elect/global` and `/etc/master-elect/KEY`
though.

## Invokation

    master-elect [OPTIONS] SERVICE

Option              |  Description                   | Default value
--------------------|--------------------------------|----------------------
--etcd-servers=[]   | etcd cluster                   | http://localhost:2379
--ttl=30            | ttl for the lock in seconds    | 30
--sleep=5           | time between checking the lock | 5
--whoami=HOSTNAME   | value to use for the lock      | the hostname
--unit=systemd-unit | systemd unit to start/monitor  | SERVICE.service
--key               | key to use for the lock        | /master-elect/SERVICE


Options need not be supplied on the command line. They can be provided
using environment variables where the option name is uppercased, `-`
replaced by `_` and `MASTER_ELECT_` or `MASTER_ELECT_SERVICE_` is
prepended. E.g. for `master-elect my-service` it will look into 
`MASTER_ELECT_TTL` and `MASTER_ELECT_MY_SERVICE_TTL` to determine the ttl.

Commandline arguments take preference over the specialised environment
variables and those take preference over the general environment variables.
