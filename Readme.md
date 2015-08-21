# Master election using etcd to start and monitor systemd units

## Master election

`leader-elect` will create a temporal lock in etcd, setting the hostname 
of the running leader. The leader will periodically reset the ttl while 
other nodes will periodically check whether the key is still there and
race to become leader when that's not the case.

The logic is inspired by kubernetes podmaster.

## Configuration

Configuration is done using the command line arguments and environment
variables. When using the provided systemd units, it will load 
configuraton from `/etc/leader-elect/global` and `/etc/leader-elect/KEY`
though.

## Invokation

    leader-elect [OPTIONS] SERVICE

Option              |  Description                   | Default value
--------------------|--------------------------------|----------------------
--etcd-servers=[]   | etcd cluster                   | http://localhost:2379
--ttl=30            | ttl for the lock in seconds    | 30
--sleep=5           | time between checking the lock | 5
--whoami=HOSTNAME   | value to use for the lock      | the hostname
--unit=systemd-unit | systemd unit to start/monitor  | SERVICE.service
--key               | key to use for the lock        | /leader-elect/SERVICE


Options need not be supplied on the command line. They can be provided
using environment variables where the option name is uppercased, `-`
replaced by `_` and `MASTER_ELECT_` or `MASTER_ELECT_SERVICE_` is
prepended. E.g. for `leader-elect my-service` it will look into 
`MASTER_ELECT_TTL` and `MASTER_ELECT_MY_SERVICE_TTL` to determine the ttl.

Commandline arguments take preference over the specialised environment
variables and those take preference over the general environment variables.

## License

leader-elect is distributed under the terms of the GPL-3. See COPYING.
If that might be a problem for you, contact me.
