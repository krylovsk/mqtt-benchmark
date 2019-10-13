MQTT benchmarking tool
=========

A simple MQTT (broker) benchmarking tool.

Installation:

```sh
go get github.com/krylovsk/mqtt-benchmark
```

The tool supports multiple concurrent clients, configurable message size, etc:

```sh
> mqtt-benchmark --help
Usage of mqtt-benchmark:
  -broker="tcp://localhost:1883": MQTT broker endpoint as scheme://host:port
  -clients=10: Number of clients to start
  -count=100: Number of messages to send per client
  -format="text": Output format: text|json
  -password="": MQTT password (empty if auth disabled)
  -qos=1: QoS for published messages
  -quiet=false : Suppress logs while running (except errors and the result)
  -size=100: Size of the messages payload (bytes)
  -topic="/test": MQTT topic for incoming message
  -username="": MQTT username (empty if auth disabled)
```

> NOTE: if `count=1` or `clients=1`, the sample standard deviation will be returned as `0` (convention due to the [lack of NaN support in JSON](https://tools.ietf.org/html/rfc4627#section-2.4))

Two output formats supported: human-readable plain text and JSON.

Example use and output:

```sh
> mqtt-benchmark --broker tcp://broker.local:1883 --count 100 --size 100 --clients 100 --qos 2 --format text
....

======= CLIENT 27 =======
Ratio:               1 (100/100)
Runtime (s):         16.396
Msg time min (ms):   9.466
Msg time max (ms):   1880.769
Msg time mean (ms):  150.193
Msg time std (ms):   201.884
Bandwidth (msg/sec): 6.099

========= TOTAL (100) =========
Total Ratio:                 1 (10000/10000)
Total Runime (sec):          16.398
Average Runtime (sec):       15.514
Msg time min (ms):           7.766
Msg time max (ms):           2034.076
Msg time mean mean (ms):     140.751
Msg time mean std (ms):      13.695
Average Bandwidth (msg/sec): 6.761
Total Bandwidth (msg/sec):   676.112
```

Similarly, in JSON:

```json
> mqtt-benchmark --broker tcp://broker.local:1883 --count 100 --size 100 --clients 100 --qos 2 --format json --quiet
{
    runs: [
        ...
        {
            "id": 61,
            "successes": 100,
            "failures": 0,
            "run_time": 16.142762197,
            "msg_tim_min": 12.798859,
            "msg_time_max": 1273.9553740000001,
            "msg_time_mean": 147.66799521,
            "msg_time_std": 152.08244221156286,
            "msgs_per_sec": 6.194726700402251
        }
    ],
    "totals": {
        "successes": 10000,
        "failures": 0,
        "total_run_time": 16.153741746,
        "avg_run_time": 15.14702422494,
        "msg_time_min": 7.852086000000001,
        "msg_time_max": 1285.241845,
        "msg_time_mean_avg": 136.4360292677,
        "msg_time_mean_std": 12.816965054355633,
        "total_msgs_per_sec": 681.0374046459865,
        "avg_msgs_per_sec": 6.810374046459865
    }
}
```
