# Description

[Prometheus]() exporter for [Claymore Dual Miner](https://github.com/nanopool/Claymore-Dual-Miner`), which collects statistics 
from json rpc provided by Claymore Miner. 

Stats being collected:

* Total Hashrate - mh/s
* Claymore Uptime in minutes
* Per GPU Hashrate kh/s
* [Ethereum]() coins found

# Installation

```
go get github.com/murat1985/claymore_exporter
```

or using Docker container:

```
git clone http://github.com/murat1985/claymore_exporeter.git
docker build . -t claymore_exporeter:local
docker run -dti claymore_exporter
```

# TODO

- WIP major cleanup
- Tests
