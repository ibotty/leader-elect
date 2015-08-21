// (c) Tobias Florek, 2015
// Distributed under the terms of the GPL-3.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/coreos/go-systemd/dbus"
)

type State struct {
	etcd    *etcd.Client
	systemd *dbus.Conn

	isMaster  bool

	etcdCluster string
	ttl         uint64
	sleep       time.Duration
	whoami      string
	unit        string
	key         string
	identifier  string
}

func IsEtcdNotFound(err error) bool {
	EcodeKeyNotFound := 100
	etcdErr, ok := err.(*etcd.EtcdError)
	return ok && err != nil && etcdErr.ErrorCode == EcodeKeyNotFound
}

func (s *State) CreateLock() (bool, error) {
	_, err := s.etcd.Create(s.key, s.whoami, s.ttl)
	if err != nil {
		return false, err
	}

	s.isMaster = true
	err = s.StartService()

	if err != nil {
		fmt.Println("Cannot start service: ", err)
		s.RemoveLock()
		return false, err
	}

	go s.MonitorLoop()

	return true, nil
}

func (s *State) RemoveLock() error {
	fmt.Println("Removing lock.")
	_, err := s.etcd.CompareAndDelete(s.key, s.whoami, 0)
	s.StopService()
	s.isMaster = false
	return err
}

func (s *State) AcquireOrRenewLock() (bool, error) {
	res, err := s.etcd.Get(s.key, false, false)
	if err != nil {
		if IsEtcdNotFound(err) {
			fmt.Println("Trying to get the lock.")
			return s.CreateLock()
		}
	}
	if res.Node.Value == s.whoami {
		if res.Node.Expiration.Sub(time.Now()) < time.Duration(s.ttl/2)*time.Second {
			_, err := s.etcd.CompareAndSwap(s.key, s.whoami, s.ttl, s.whoami, 0)
			if err != nil {
				fmt.Println("Could not renew the lock: ", err)
			}
		}
		return true, nil
	}

	return false, nil
}

func (s *State) StartService() error {
	// the following works in 3.0
	//startup := make(chan string)
	//_, err := s.systemd.StartUnit(s.unit, "fail", startup)
	//if err != nil {
	//     return err
	//}
	//unitStatus := <- startup

	// this one in 2.x
	unitStatus, err := s.systemd.StartUnit(s.unit, "fail")
	if err != nil {
		return err
	}

	switch unitStatus {
	case "done":
		fmt.Println("Service started successfully.")
	default:
		return errors.New(unitStatus)
	}
	return nil
}

func (s *State) StopService() error {
	// the following works in 3.0
	//startup := make(chan string)
	//_, err := s.systemd.StopUnit(s.unit, "fail", startup)

	// this one in 2.x
	_, err := s.systemd.StopUnit(s.unit, "fail")

	return err
}

func (s *State) LockLoop() {
	for {
		leader, err := s.AcquireOrRenewLock()
		switch {
		case leader:
			if err != nil {
				fmt.Println("Cannot start service. Releasing lock. ", err)
				s.RemoveLock()
			}
		default:
		}
		time.Sleep(s.sleep)
	}
}

func (s *State) MonitorLoop() {
	fmt.Println("Starting monitoring loop.")
	ch, errCh := s.systemd.SubscribeUnitsCustom(s.sleep, 0, func(a, b *dbus.UnitStatus) bool { return *a != *b }, func(u string) bool { return u != s.unit })

	for {
		// not leader anymore. Don't monitor anymore.
		if !s.isMaster {
			return
		}

		select {
		case uss := <-ch:
			us, ok := uss[s.unit]
			if !ok || us.ActiveState == "failed" {
				fmt.Println("unit changed state: ", us)
				s.RemoveLock()
			}
		case err := <-errCh:
			fmt.Println("Error while monitoring systemd unit: ", err)
			s.RemoveLock()
		}
	}
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

func SetupFlags(s *State) *flag.FlagSet {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Cannot get hostname: ", err)
		os.Exit(2)
	}

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fs.StringVar(&s.etcdCluster, "etcd-servers", "http://localhost:2379", "Comma-separated list of etcd servers to use")
	fs.Uint64Var(&s.ttl, "ttl", 30, "time to live for the lock in seconds")
	fs.DurationVar(&s.sleep, "sleep", 5*time.Second, "time between checking the lock in seconds")
	fs.StringVar(&s.whoami, "whoami", hostname, "value to identify the leader-elect instance")
	fs.StringVar(&s.unit, "unit", "", "Systemd unit to start/monitor")
	fs.StringVar(&s.key, "key", "", "Etcd key to use for the lock")

	return fs
}

func main() {
	s := State{}
	fs := SetupFlags(&s)
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println("Cannot parse: ", err)
		os.Exit(1)
	}
	if len(fs.Args()) != 1 {
		fmt.Println("You need to supply a name")
		os.Exit(1)
	}
	s.identifier = fs.Arg(0)
	AddEnvironmentToFlags("MASTER_ELECT_"+ToEnvironmentKey(s.identifier), fs)
	AddEnvironmentToFlags("MASTER_ELECT", fs)
	if s.unit == "" {
		s.unit = s.identifier + ".service"
	}
	if s.key == "" {
		s.key = "/leader-elect.ibotty.net/" + s.identifier
	}

	fmt.Println("Starting with config: ", s)

	conn, err := dbus.New()
	if err != nil {
		fmt.Println("Cannot connect to systemd: ", err)
		os.Exit(2)
	}
	s.systemd = conn
	err = s.systemd.Subscribe()

	if err != nil {
		fmt.Println("Cannot subscribe to unit changes: ", err)
		os.Exit(2)
	}

	etcd := etcd.NewClient(strings.Split(s.etcdCluster, ","))
	s.etcd = etcd

	s.LockLoop()
}
