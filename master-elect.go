// (c) Tobias Florek, 2015
// Distributed under the terms of the GPL-3.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	etcdCluster string
	ttl         uint64
	sleep       time.Duration
	whoami      string
	unit        string
	key         string
	identifier  string
}

// from github.com/coreos/etcd
//
// SetFlagsFromEnv parses all registered flags in the given flagset,
// and if they are not already set it attempts to set their values from
// environment variables. Environment variables take the name of the flag but
// are UPPERCASE, have the prefix `s`, and any dashes are replaced by
// underscores - for example: some-flag => ETCD_SOME_FLAG
func AddEnvironmentToFlags(s string, fs *flag.FlagSet) error {
	var err error
	alreadySet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		alreadySet[f.Name] = true
	})
	fs.VisitAll(func(f *flag.Flag) {
		if !alreadySet[f.Name] {
			key := s + "_" + ToEnvironmentKey(f.Name)
			val := os.Getenv(key)
			if val != "" {
				if serr := fs.Set(f.Name, val); serr != nil {
					err = fmt.Errorf("invalid value %q for %s: %v", val, key, serr)
				}
			}
		}
	})
	return err
}

func ToEnvironmentKey(s string) string {
	return strings.ToUpper(strings.Replace(s, "-", "_", -1))
}

func SetupFlags(c *Config) *flag.FlagSet {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Cannot get hostname: ", err)
		os.Exit(1)
	}

	fs := flag.NewFlagSet(os.Args[0], flag.PanicOnError)

	fs.StringVar(&c.etcdCluster, "etcd-servers", "http://localhost:2179", "Comma-separated list of etcd servers to use")
	fs.Uint64Var(&c.ttl, "ttl", 30, "time to live for the lock in seconds")
	fs.DurationVar(&c.sleep, "sleep", 5*time.Second, "time between checking the lock in seconds")
	fs.StringVar(&c.whoami, "whoami", hostname, "value to identify the master-elect instance")
	fs.StringVar(&c.unit, "unit", "", "Systemd unit to start/monitor")
	fs.StringVar(&c.key, "key", "", "Etcd key to use for the lock")

	return fs
}

func main() {
	c := Config{}
	fs := SetupFlags(&c)
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println("Cannot parse: ", err)
		os.Exit(1)
	}
	if len(fs.Args()) != 1 {
		fmt.Println("You need to supply a name")
		os.Exit(1)
	}
	c.identifier = fs.Arg(0)
	AddEnvironmentToFlags("MASTER_ELECT_"+ToEnvironmentKey(c.identifier), fs)
	AddEnvironmentToFlags("MASTER_ELECT", fs)
	if c.unit == "" {
		c.unit = c.identifier + ".service"
	}
	if c.key == "" {
		c.key = c.identifier
	}

	fmt.Println("config:", c)
}
