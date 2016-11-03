# Can I use afpacket fanout?

    $ go build && sudo ./can-i-use-afpacket-fanout  -interface wlan0
    2016/11/02 21:02:41 Starting worker id 0 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 1 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 2 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 3 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 4 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 5 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 6 on interface wlan0
    2016/11/02 21:02:41 Starting worker id 7 on interface wlan0
    2016/11/02 21:02:56 packets=100 flows=9 success=152 failures=0 reverse_failures=0
    2016/11/02 21:03:03 packets=200 flows=23 success=287 failures=0 reverse_failures=0
    2016/11/02 21:03:07 packets=300 flows=33 success=440 failures=0 reverse_failures=0
    2016/11/02 21:03:11 packets=400 flows=41 success=602 failures=0 reverse_failures=0
    2016/11/02 21:03:13 packets=500 flows=42 success=785 failures=0 reverse_failures=0
    2016/11/02 21:03:13 packets=600 flows=68 success=914 failures=0 reverse_failures=0
    2016/11/02 21:03:13 packets=700 flows=80 success=1098 failures=0 reverse_failures=0
    2016/11/02 21:03:14 packets=800 flows=84 success=1290 failures=0 reverse_failures=0
    2016/11/02 21:03:14 packets=900 flows=92 success=1475 failures=0 reverse_failures=0
    2016/11/02 21:03:14 packets=1000 flows=92 success=1675 failures=0 reverse_failures=0
    2016/11/02 21:03:14 packets=1100 flows=92 success=1873 failures=0 reverse_failures=0
    2016/11/02 21:03:14 packets=1119 flows=101 success=1894 failures=0 reverse_failures=0
    2016/11/02 21:03:14 Worker flow count distribution:
    2016/11/02 21:03:14 worker=3 flows=16
    2016/11/02 21:03:14 worker=6 flows=10
    2016/11/02 21:03:14 worker=1 flows=12
    2016/11/02 21:03:14 worker=2 flows=7
    2016/11/02 21:03:14 worker=0 flows=13
    2016/11/02 21:03:14 worker=7 flows=17
    2016/11/02 21:03:14 worker=5 flows=16
    2016/11/02 21:03:14 worker=4 flows=10



YES!
