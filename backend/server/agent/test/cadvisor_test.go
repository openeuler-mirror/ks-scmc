package test

import (
	"testing"
	"time"

	"github.com/google/cadvisor/client"
	v1 "github.com/google/cadvisor/info/v1"
)

func TestCadvisor(t *testing.T) {
	client, err := client.NewClient("http://172.17.0.6:8080/")
	if err != nil {
		t.Error(err)
	}

	minfo, err := client.MachineInfo()
	if err != nil {
		t.Error(err)
	}

	t.Logf("CPU cores:%v Memory:%vMB", minfo.NumCores, minfo.MemoryCapacity/(1<<20))

	now := time.Now()
	request := v1.ContainerInfoRequest{
		// NumStats: 5,
		Start: now.Add(-time.Second * 10),
		End:   now,
	}
	// t.Logf("%+v", request)
	infos, err := client.AllDockerContainers(&request)
	if err != nil {
		t.Error(err)
	}

	for _, info := range infos {
		if len(info.Stats) > 1 {
			head := info.Stats[0]
			tail := info.Stats[len(info.Stats)-1]

			duration := tail.Timestamp.Sub(head.Timestamp)
			percent := float64(tail.Cpu.Usage.Total-head.Cpu.Usage.Total) / float64(duration.Nanoseconds())
			t.Logf("%v CPU=%.3f%% memory=%.3fMB", info.Aliases[0], percent*100, float64(tail.Memory.WorkingSet)/(1<<20))
		}
	}
}

func TestCadvisor1(t *testing.T) {
	client, err := client.NewClient("http://172.17.0.2:8080/")
	if err != nil {
		t.Error(err)
	}

	// now := time.Now()
	request := v1.ContainerInfoRequest{
		// NumStats: 5,
		// Start: now.Add(-time.Second * 10),
		// End:   now,
	}
	// t.Logf("%+v", request)
	infos, err := client.SubcontainersInfo("/", &request)
	if err != nil {
		t.Error(err)
	}

	for _, info := range infos {
		if info.Name == "/" {
			if len(info.Stats) > 1 {
				head := info.Stats[0]
				tail := info.Stats[len(info.Stats)-1]

				duration := tail.Timestamp.Sub(head.Timestamp)
				percent := float64(tail.Cpu.Usage.Total-head.Cpu.Usage.Total) / float64(duration.Nanoseconds())
				t.Logf("%v CPU=%.3f%% memory=%.3fMB", info.Aliases, percent*100, float64(tail.Memory.WorkingSet)/(1<<20))
			}
		}
		// t.Logf("id=%v name=%v", info.ContainerReference.Id, info.ContainerReference.Name)
	}
}
