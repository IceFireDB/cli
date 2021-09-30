## quick start
> this is a sample about how to start up 2 icefiredb group, and manage the slot.

### start etcd(or zookeeper)

### start icefire
```shell
// clone the project
make
./bin/IceFireDB -a 0.0.0.0:6399 --coordinator-addr http://localhost:2379 --coordinator-type etcd --announce-ip 127.0.0.1 --announce-port 6399 -d ./testdata/6399 &

./bin/IceFireDB -a 0.0.0.0:6398 --coordinator-addr http://localhost:2379 --coordinator-type etcd --announce-ip 127.0.0.1 --announce-port 6398 -d ./testdata/6398 &
```


### start slot manege
> Follow the steps in samples. 
```shell
make
cd samples
./add_group.sh
./initslot.sh
```

### start proxy
```shell
// clone the project and modify config.sample.ini
make

./bin/proxy -c config.sample.ini -L ./log/proxy.log  --addr=0.0.0.0:19000 --http-addr=0.0.0.0:11000 &
```

### test
> the sample here used key `49`, because it is located in the slot 0. The migrating script is using slot 0 as example.

```shell
# redis-cli -p 6398
127.0.0.1:6398> get 49
"sdfsdf"
# redis-cli -p 6399
127.0.0.1:6399> get 49
""
// run migrate sh, 
# redis-cli -p 6399
127.0.0.1:6399> get 49
"sdfsdf"
# redis-cli -p 6398
127.0.0.1:6398> get 49
""
```

At the meanwhile, you can always get the value from proxy:
```shell
# redis-cli -p 19000
127.0.0.1:19000> get 49
"sdfsdf"
```
