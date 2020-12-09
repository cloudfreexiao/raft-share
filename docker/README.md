https://juejin.im/post/6844904031186321416

docker exec -t -i docker_etcd_1 /bin/bash

etcd --version
etcdctl -h
etcdctl put /testdir/testkey "Hello world"
etcdctl get /testdir/testkey