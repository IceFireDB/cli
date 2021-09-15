0. start zookeeper or etcd 
1. modify config.ini modify coordinator_type and coordinator_addr change other config items in config.ini
2. start 2 icefiredb instance for example listening 6398 and 6399
3. ./add_group.sh
4. ./initslot.sh

