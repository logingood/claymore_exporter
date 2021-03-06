# Description

[Prometheus]() exporter for [Claymore Dual Miner](https://github.com/nanopool/Claymore-Dual-Miner`), which collects statistics 
from json rpc provided by Claymore Miner. 

Stats being collected:

* Total Hashrate - mh/s
* Claymore Uptime in minutes
* Per GPU Hashrate kh/s
* [Ethereum](https://www.ethereum.org/) shares found
* Rejected shares

# Installation

```
go get github.com/murat1985/claymore_exporter
```

or using Docker container:

```
git clone http://github.com/murat1985/claymore_exporeter.git
```

Build local image:
```
docker build . -t claymore_exporeter:local
```

Run it:
```
docker run -d -t -i -e CLAYMORE_DIAL_ADDR='192.168.1.1;192.168.1.2;192.168.1.3' -p 10333:10333 --name claymore_exporter claymore_exporter:local
```

# TODO

- WIP major cleanup
- Tests
