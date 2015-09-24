#!/bin/sh

ovs-vsctl show | grep Bridge | awk '{print $2}' | xargs -n1 ovs-vsctl del-br
ip netns list | xargs  -n1 ip netns del
ip link | egrep -oh "[0-9]+: .*+:" | grep -v eth0 | awk -F ':' '{print $2}' | xargs -n1 ip link delete
