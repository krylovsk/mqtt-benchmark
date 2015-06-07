MQTT benchmarking tool
=========
A simple MQTT (broker) benchmarking tool.

Supports multiple concurrent clients, configurable message size, etc:
```
> mqtt-benchmark --help
Usage of mqtt-benchmark:
  -broker="tcp://localhost:1883": MQTT broker endpoint as scheme://host:port
  -clients=10: Number of clients to start
  -count=100: Number of messages to send per client
  -format="text": Output format: text|json
  -password="": MQTT password (empty if auth disabled)
  -qos=1: QoS for published messages
  -size=100: Size of the messages payload (bytes)
  -topic="/test": MQTT topic for incoming message
  -username="": MQTT username (empty if auth disabled)
```

Two output formats supported: human-readable plain text and JSON.

Example use and output:

```
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