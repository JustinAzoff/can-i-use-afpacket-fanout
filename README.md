# Can I use afpacket fanout?

## Background

The linux kernel has a feature for efficiently capturing packets called afpacket.

A related feature,  `fanout groups`, exists so that you can capture from N proccesses or threads at the
same time, and each thread will see 1/Nth of the traffic.  For stateful
applications that perform stream reassembly, instead of a simple round robin distribution the packets need to
be hashed according to their 5 tuple.  To further complicate the issue, the hash function
needs to be symmetrical so that a packet from HostA to HostB is hashed to the
same proccess as the packet from HostB to HostA.  This feature is known as PACKET\_FANOUT\_HASH

Unfortnately, some versions of the linux kernel are broken and do not properly
implement the symmetric hash.  [This issue has been
fixed](https://git.kernel.org/cgit/linux/kernel/git/davem/net-next.git/commit/?id=eb70db8756717b90c01ccc765fdefc4dd969fc74),
but the buggy code made its way into various distribution kernels.

It's not easy to know just by looking at a kernel version whether or not it
will work properly.

`can-i-use-afpacket-fanout` is a tool that runs multiple threads in a fanout group and checks to
see if flows are routed to the appropriate workers.  If it sees a flow
on two different workers, or the reverse flow on a different worker, it
will log a FAILure.

## Install

    $ export GOPATH=~/go # If you don't already have this set to something
    $ go get github.com/JustinAzoff/can-i-use-afpacket-fanout

## RUN

    $ sudo ~/go/bin/can-i-use-afpacket-fanout -interface wlan0 -maxflows 500
    2016/11/08 03:55:20 Starting worker id 0 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 1 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 2 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 3 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 4 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 5 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 6 on interface wlan0
    2016/11/08 03:55:20 Starting worker id 7 on interface wlan0
    2016/11/08 03:55:20 Collecting results until 500 flows have been seen..
    2016/11/08 03:55:22 Stats: packets=100 flows=31 success=69 reverse_success=83 failures=0 reverse_failures=0
    2016/11/08 03:55:23 Stats: packets=200 flows=42 success=158 reverse_success=176 failures=0 reverse_failures=0
    2016/11/08 03:55:24 Stats: packets=300 flows=72 success=228 reverse_success=258 failures=0 reverse_failures=0
    2016/11/08 03:55:25 Stats: packets=400 flows=87 success=313 reverse_success=348 failures=0 reverse_failures=0
    2016/11/08 03:55:27 Stats: packets=500 flows=123 success=377 reverse_success=425 failures=0 reverse_failures=0
    2016/11/08 03:55:29 Stats: packets=600 flows=151 success=449 reverse_success=506 failures=0 reverse_failures=0
    2016/11/08 03:55:31 Stats: packets=700 flows=179 success=521 reverse_success=588 failures=0 reverse_failures=0
    2016/11/08 03:55:33 Stats: packets=800 flows=202 success=598 reverse_success=670 failures=0 reverse_failures=0
    2016/11/08 03:55:34 Stats: packets=900 flows=227 success=673 reverse_success=755 failures=0 reverse_failures=0
    2016/11/08 03:55:36 Stats: packets=1000 flows=247 success=753 reverse_success=838 failures=0 reverse_failures=0
    2016/11/08 03:55:37 Stats: packets=1100 flows=269 success=831 reverse_success=926 failures=0 reverse_failures=0
    2016/11/08 03:55:38 Stats: packets=1200 flows=291 success=909 reverse_success=1007 failures=0 reverse_failures=0
    2016/11/08 03:55:40 Stats: packets=1300 flows=305 success=995 reverse_success=1094 failures=0 reverse_failures=0
    2016/11/08 03:55:41 Stats: packets=1400 flows=325 success=1075 reverse_success=1176 failures=0 reverse_failures=0
    2016/11/08 03:55:43 Stats: packets=1500 flows=346 success=1154 reverse_success=1263 failures=0 reverse_failures=0
    2016/11/08 03:55:44 Stats: packets=1600 flows=372 success=1228 reverse_success=1345 failures=0 reverse_failures=0
    2016/11/08 03:55:45 Stats: packets=1700 flows=386 success=1314 reverse_success=1431 failures=0 reverse_failures=0
    2016/11/08 03:55:46 Stats: packets=1800 flows=399 success=1401 reverse_success=1519 failures=0 reverse_failures=0
    2016/11/08 03:55:46 Stats: packets=1900 flows=409 success=1491 reverse_success=1608 failures=0 reverse_failures=0
    2016/11/08 03:55:48 Stats: packets=2000 flows=431 success=1569 reverse_success=1693 failures=0 reverse_failures=0
    2016/11/08 03:55:49 Stats: packets=2100 flows=448 success=1652 reverse_success=1778 failures=0 reverse_failures=0
    2016/11/08 03:55:50 Stats: packets=2200 flows=470 success=1730 reverse_success=1864 failures=0 reverse_failures=0
    2016/11/08 03:55:51 Stats: packets=2300 flows=493 success=1807 reverse_success=1945 failures=0 reverse_failures=0
    2016/11/08 03:55:52 Final Stats: packets=2345 flows=501 success=1844 reverse_success=1983 failures=0 reverse_failures=0
    2016/11/08 03:55:52 Worker flow count distribution:
    2016/11/08 03:55:52  - worker=0 flows=54
    2016/11/08 03:55:52  - worker=1 flows=79
    2016/11/08 03:55:52  - worker=2 flows=44
    2016/11/08 03:55:52  - worker=3 flows=58
    2016/11/08 03:55:52  - worker=4 flows=71
    2016/11/08 03:55:52  - worker=5 flows=73
    2016/11/08 03:55:52  - worker=6 flows=64
    2016/11/08 03:55:52  - worker=7 flows=58


YES!
