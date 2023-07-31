# KubeArmor PodInformer Implementation

Result:

```
$ go run main.go
INFO[0000] Initial containers: [mynginx coredns coredns etcd kindnet-cni kube-apiserver kube-controller-manager kube-proxy kube-scheduler local-path-provisioner] 
INFO[0000] Starting Pod controller
INFO[0000] New containers: coredns
INFO[0000] New containers: mynginx
INFO[0000] New containers: kube-controller-manager
INFO[0000] New containers: etcd
INFO[0000] New containers: kube-scheduler
INFO[0000] New containers: kube-proxy
INFO[0000] New containers: kindnet-cni
INFO[0000] New containers: coredns
INFO[0000] New containers: local-path-provisioner
INFO[0000] New containers: kube-apiserver
INFO[0020] New containers: mynginx
INFO[0021] New containers: mynginx
INFO[0021] New containers: mynginx
INFO[0021] Pod with key default/mynginx delete
INFO[0233] New containers: busybox
INFO[0233] New containers: busybox
INFO[0233] Pod with key default/busybox delete
```
