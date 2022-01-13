package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func isRealNetwork(name string) bool {
	if name == "" {
		return false
	}

	/*
	 * If /sys/class/net/<interface>/device exists, <interface> is a real network
	 * according to `man 5 sysfs`
	 * /sys/class/net
	 *        Each of the entries in this directory is a symbolic link
	 *        representing one of the real or virtual networking devices
	 *        that are visible in the network namespace of the process
	 *        that is accessing the directory.  Each of these symbolic
	 *        links refers to entries in the /sys/devices directory.
	 */
	const networkDevicePattern = "/sys/class/net/%s/device"
	path := fmt.Sprintf(networkDevicePattern, name)
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("os.Stat %s: %v", path, err)
		}
		return false
	}

	return true
}

func TestListInterfaces(t *testing.T) {
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Error(err)
	}

	var ipnet net.IPNet

	for _, i := range interfaces {
		if isRealNetwork(i.Name) {
			t.Logf("Name=%v Up=%v MAC=%v", i.Name, i.Flags&net.FlagUp, i.HardwareAddr)
			// t.Logf("%+v", i)
			addrs, err := i.Addrs()
			if err != nil {
				t.Error(err)
			}
			for _, a := range addrs {
				if a.Network() == ipnet.Network() {
					addr, ok := a.(*net.IPNet)
					if ok {
						ones, bits := addr.Mask.Size()
						t.Logf("\tIP=%v Gateway=%v %v %v", addr.IP, addr.Mask, ones, bits)
						// t.Log(addr.IP.String(), addr.Mask.String())
					}
				} else {
					t.Logf("\t%v %#v", a.Network(), a)
				}

			}

			addrs, err = i.MulticastAddrs()
			if err != nil {
				t.Error(err)
			}
			for _, a := range addrs {
				t.Logf("\t\t%v %v", a.Network(), a)
			}
		}
	}
}

func TestDockerInterface(t *testing.T) {
	/*
		docker network create \
			  --driver=bridge \
			  --subnet=172.28.0.0/16 \
			  --ip-range=172.28.5.0/24 \
			  --gateway=172.28.5.254 \
			  br0
	*/
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Errorf("init docker client: %v", err)
	}

	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		t.Errorf("NetworkList: %v", err)
	}

	for _, n := range list {
		if n.Driver != "bridge" {
			continue
		}
		t.Logf("%v %v %+v", n.Name, n.Driver, n.IPAM)
	}
}
