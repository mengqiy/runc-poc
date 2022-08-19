package main

import (
	"context"
	"encoding/json"
	"os"
	"runtime"

	"github.com/mengqiy/runc-poc/images"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.Info("os.Args", os.Args)

	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, _ := libcontainer.New("")
		if err := factory.StartInitialization(); err != nil {
			logrus.Fatal(err)
		}
		panic("--this line should have never been executed, congratulations--")
	}
}

func main() {
	// var inConfigFile = flag.String("config-file", "config.json", "input config file")
	// flag.Parse()

	// logrus.Info(*inConfigFile)

	store, err := images.NewStore("/usr/local/google/home/mengqiy/.cache/runm")
	if err != nil {
		logrus.Fatal(err)
		return
	}
	// imgs, err := store.Pull("alpine:3.15.0", "linux", "amd64")
	// if err != nil {
	// 	logrus.Fatal(err)
	// 	return
	// }
	extractedImage, err := store.Extract(context.Background(), "gcr.io/kpt-fn/gatekeeper:v0")
	if err != nil {
		logrus.Fatal(err)
		return
	}
	factory, err := libcontainer.New(extractedImage.ExtractedDir)

	logrus.Info(extractedImage.ExtractedDir)

	// factory, err := libcontainer.New("/usr/local/google/home/mengqiy/mycontainer")
	// factory, err := libcontainer.New("/mycontainer")
	if err != nil {
		logrus.Fatal(err)
		return
	}

	// var devicesRules []*devices.Rule
	// for _, device := range specconv.AllowedDevices {
	// 	devicesRules = append(devicesRules, &device.Rule)
	// }
	// config := GetConfig(devicesRules)

	// config, err := getConfig(false, extractedImage.ExtractedDir)
	config, err := getConfig(extractedImage.ExtractedDir)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	container, err := factory.Create("mycontainerid", config)
	if err != nil {
		logrus.Fatal(err)
		return
	}

	process := &libcontainer.Process{
		Args:   []string{"/bin/echo", "helloworld"},
		Env:    []string{"PATH=/bin"},
		User:   "root",
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Init:   true,
	}

	err = container.Run(process)
	if err != nil {
		container.Destroy()
		logrus.Fatal(err)
		return
	}

	// wait for the process to finish.
	_, err = process.Wait()
	if err != nil {
		logrus.Fatal(err)
	}

	// destroy the container.
	container.Destroy()
}

func getConfig(rootfs string) (*configs.Config, error) {
	// in, err := ioutil.ReadFile(inputPath)
	// if err != nil {
	// 	return nil, err
	// }
	var config configs.Config
	err := json.Unmarshal([]byte(rootlessConfigJson), &config)
	if err != nil {
		return nil, err
	}
	config.Rootfs = rootfs
	return &config, nil
}

// func getConfig(rootful bool, rootfs string) (*configs.Config, error) {
// 	var cfg string
// 	if rootful {
// 		cfg = rootfulConfigJson
// 	} else {
// 		cfg = rootlessConfigJson
// 	}
// 	var config configs.Config
// 	err := json.Unmarshal([]byte(cfg), &config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	config.Rootfs = rootfs
// 	return &config, nil
// }

// Rootless docker use the following config when running with runc.
// This config can be converted to a golang object using the struct defined in libcontainer.
const rootlessConfigJson = `{
   "no_pivot_root":false,
   "parent_death_signal":0,
   "umask":null,
   "readonlyfs":true,
   "rootPropagation":0,
   "mounts":[
      {
         "source":"proc",
         "destination":"/proc",
         "device":"proc",
         "flags":0,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"tmpfs",
         "destination":"/dev",
         "device":"tmpfs",
         "flags":16777218,
         "propagation_flags":null,
         "data":"mode=755,size=65536k",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"devpts",
         "destination":"/dev/pts",
         "device":"devpts",
         "flags":10,
         "propagation_flags":null,
         "data":"newinstance,ptmxmode=0666,mode=0620",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"shm",
         "destination":"/dev/shm",
         "device":"tmpfs",
         "flags":14,
         "propagation_flags":null,
         "data":"mode=1777,size=65536k",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"mqueue",
         "destination":"/dev/mqueue",
         "device":"mqueue",
         "flags":14,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"/sys",
         "destination":"/sys",
         "device":"bind",
         "flags":20495,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"cgroup",
         "destination":"/sys/fs/cgroup",
         "device":"cgroup",
         "flags":2097167,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      }
   ],
   "devices":[
      {
         "type":99,
         "major":1,
         "minor":3,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/null",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      },
      {
         "type":99,
         "major":1,
         "minor":8,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/random",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      },
      {
         "type":99,
         "major":1,
         "minor":7,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/full",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      },
      {
         "type":99,
         "major":5,
         "minor":0,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/tty",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      },
      {
         "type":99,
         "major":1,
         "minor":5,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/zero",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      },
      {
         "type":99,
         "major":1,
         "minor":9,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/urandom",
         "file_mode":438,
         "uid":405368,
         "gid":89939
      }
   ],
   "mount_label":"",
   "hostname":"runc",
   "namespaces":[
      {
         "type":"NEWPID",
         "path":""
      },
      {
         "type":"NEWIPC",
         "path":""
      },
      {
         "type":"NEWUTS",
         "path":""
      },
      {
         "type":"NEWNS",
         "path":""
      },
      {
         "type":"NEWCGROUP",
         "path":""
      },
      {
         "type":"NEWUSER",
         "path":""
      }
   ],
   "capabilities":{
      "Bounding":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Effective":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Inheritable":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Permitted":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Ambient":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ]
   },
   "networks":null,
   "routes":null,
   "cgroups":{
      "name":"mycontainerid",
      "path":"",
      "scope_prefix":"",
      "devices":[
         {
            "type":99,
            "major":-1,
            "minor":-1,
            "permissions":"m",
            "allow":true
         },
         {
            "type":98,
            "major":-1,
            "minor":-1,
            "permissions":"m",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":3,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":8,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":7,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":5,
            "minor":0,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":5,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":9,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":136,
            "minor":-1,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":5,
            "minor":2,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":10,
            "minor":200,
            "permissions":"rwm",
            "allow":true
         }
      ],
      "memory":0,
      "memory_reservation":0,
      "memory_swap":0,
      "cpu_shares":0,
      "cpu_quota":0,
      "cpu_period":0,
      "cpu_rt_quota":0,
      "cpu_rt_period":0,
      "cpuset_cpus":"",
      "cpuset_mems":"",
      "pids_limit":0,
      "blkio_weight":0,
      "blkio_leaf_weight":0,
      "blkio_weight_device":null,
      "blkio_throttle_read_bps_device":null,
      "blkio_throttle_write_bps_device":null,
      "blkio_throttle_read_iops_device":null,
      "blkio_throttle_write_iops_device":null,
      "freezer":"",
      "hugetlb_limit":null,
      "oom_kill_disable":false,
      "memory_swappiness":null,
      "net_prio_ifpriomap":null,
      "net_cls_classid_u":0,
      "rdma":null,
      "cpu_weight":0,
      "unified":null,
      "Systemd":false,
      "Rootless":true
   },
   "uid_mappings":[
      {
         "container_id":0,
         "host_id":405368,
         "size":1
      }
   ],
   "gid_mappings":[
      {
         "container_id":0,
         "host_id":89939,
         "size":1
      }
   ],
   "mask_paths":[
      "/proc/acpi",
      "/proc/asound",
      "/proc/kcore",
      "/proc/keys",
      "/proc/latency_stats",
      "/proc/timer_list",
      "/proc/timer_stats",
      "/proc/sched_debug",
      "/sys/firmware",
      "/proc/scsi"
   ],
   "readonly_paths":[
      "/proc/bus",
      "/proc/fs",
      "/proc/irq",
      "/proc/sys",
      "/proc/sysrq-trigger"
   ],
   "sysctl":null,
   "seccomp":null,
   "no_new_privileges":true,
   "Hooks":{
      "createContainer":null,
      "createRuntime":null,
      "poststart":null,
      "poststop":null,
      "prestart":null,
      "startContainer":null
   },
   "version":"1.0.2-dev",
   "no_new_keyring":false,
   "rootless_euid":true,
   "rootless_cgroups":true
}
`


// Rootful docker use the following config when running with runc.
// This config can be converted to a golang object using the struct defined in libcontainer.
const rootfulConfigJson = `{
   "no_pivot_root":false,
   "parent_death_signal":0,
   "umask":null,
   "readonlyfs":true,
   "rootPropagation":0,
   "mounts":[
      {
         "source":"proc",
         "destination":"/proc",
         "device":"proc",
         "flags":0,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"tmpfs",
         "destination":"/dev",
         "device":"tmpfs",
         "flags":16777218,
         "propagation_flags":null,
         "data":"mode=755,size=65536k",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"devpts",
         "destination":"/dev/pts",
         "device":"devpts",
         "flags":10,
         "propagation_flags":null,
         "data":"newinstance,ptmxmode=0666,mode=0620,gid=5",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"shm",
         "destination":"/dev/shm",
         "device":"tmpfs",
         "flags":14,
         "propagation_flags":null,
         "data":"mode=1777,size=65536k",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"mqueue",
         "destination":"/dev/mqueue",
         "device":"mqueue",
         "flags":14,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"sysfs",
         "destination":"/sys",
         "device":"sysfs",
         "flags":15,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      },
      {
         "source":"cgroup",
         "destination":"/sys/fs/cgroup",
         "device":"cgroup",
         "flags":2097167,
         "propagation_flags":null,
         "data":"",
         "relabel":"",
         "rec_attr":null,
         "extensions":0,
         "premount_cmds":null,
         "postmount_cmds":null
      }
   ],
   "devices":[
      {
         "type":99,
         "major":1,
         "minor":3,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/null",
         "file_mode":438,
         "uid":0,
         "gid":0
      },
      {
         "type":99,
         "major":1,
         "minor":8,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/random",
         "file_mode":438,
         "uid":0,
         "gid":0
      },
      {
         "type":99,
         "major":1,
         "minor":7,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/full",
         "file_mode":438,
         "uid":0,
         "gid":0
      },
      {
         "type":99,
         "major":5,
         "minor":0,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/tty",
         "file_mode":438,
         "uid":0,
         "gid":0
      },
      {
         "type":99,
         "major":1,
         "minor":5,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/zero",
         "file_mode":438,
         "uid":0,
         "gid":0
      },
      {
         "type":99,
         "major":1,
         "minor":9,
         "permissions":"rwm",
         "allow":true,
         "path":"/dev/urandom",
         "file_mode":438,
         "uid":0,
         "gid":0
      }
   ],
   "mount_label":"",
   "hostname":"runc",
   "namespaces":[
      {
         "type":"NEWPID",
         "path":""
      },
      {
         "type":"NEWNET",
         "path":""
      },
      {
         "type":"NEWIPC",
         "path":""
      },
      {
         "type":"NEWUTS",
         "path":""
      },
      {
         "type":"NEWNS",
         "path":""
      },
      {
         "type":"NEWCGROUP",
         "path":""
      }
   ],
   "capabilities":{
      "Bounding":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Effective":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Inheritable":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Permitted":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ],
      "Ambient":[
         "CAP_AUDIT_WRITE",
         "CAP_KILL",
         "CAP_NET_BIND_SERVICE"
      ]
   },
   "networks":[
      {
         "type":"loopback",
         "name":"",
         "bridge":"",
         "mac_address":"",
         "address":"",
         "gateway":"",
         "ipv6_address":"",
         "ipv6_gateway":"",
         "mtu":0,
         "txqueuelen":0,
         "host_interface_name":"",
         "hairpin_mode":false
      }
   ],
   "routes":null,
   "cgroups":{
      "name":"mycontainerid",
      "path":"",
      "scope_prefix":"",
      "devices":[
         {
            "type":97,
            "major":-1,
            "minor":-1,
            "permissions":"rwm",
            "allow":false
         },
         {
            "type":99,
            "major":-1,
            "minor":-1,
            "permissions":"m",
            "allow":true
         },
         {
            "type":98,
            "major":-1,
            "minor":-1,
            "permissions":"m",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":3,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":8,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":7,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":5,
            "minor":0,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":5,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":1,
            "minor":9,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":136,
            "minor":-1,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":5,
            "minor":2,
            "permissions":"rwm",
            "allow":true
         },
         {
            "type":99,
            "major":10,
            "minor":200,
            "permissions":"rwm",
            "allow":true
         }
      ],
      "memory":0,
      "memory_reservation":0,
      "memory_swap":0,
      "cpu_shares":0,
      "cpu_quota":0,
      "cpu_period":0,
      "cpu_rt_quota":0,
      "cpu_rt_period":0,
      "cpuset_cpus":"",
      "cpuset_mems":"",
      "pids_limit":0,
      "blkio_weight":0,
      "blkio_leaf_weight":0,
      "blkio_weight_device":null,
      "blkio_throttle_read_bps_device":null,
      "blkio_throttle_write_bps_device":null,
      "blkio_throttle_read_iops_device":null,
      "blkio_throttle_write_iops_device":null,
      "freezer":"",
      "hugetlb_limit":null,
      "oom_kill_disable":false,
      "memory_swappiness":null,
      "net_prio_ifpriomap":null,
      "net_cls_classid_u":0,
      "rdma":null,
      "cpu_weight":0,
      "unified":null,
      "Systemd":false,
      "Rootless":false
   },
   "uid_mappings":null,
   "gid_mappings":null,
   "mask_paths":[
      "/proc/acpi",
      "/proc/asound",
      "/proc/kcore",
      "/proc/keys",
      "/proc/latency_stats",
      "/proc/timer_list",
      "/proc/timer_stats",
      "/proc/sched_debug",
      "/sys/firmware",
      "/proc/scsi"
   ],
   "readonly_paths":[
      "/proc/bus",
      "/proc/fs",
      "/proc/irq",
      "/proc/sys",
      "/proc/sysrq-trigger"
   ],
   "sysctl":null,
   "seccomp":null,
   "no_new_privileges":true,
   "Hooks":{
      "createContainer":null,
      "createRuntime":null,
      "poststart":null,
      "poststop":null,
      "prestart":null,
      "startContainer":null
   },
   "version":"1.0.2-dev",
   "no_new_keyring":false
}
`
