# Hostile attacks

This repository contains protocols of hostile attacks.

Once a day in the morning the data of the previous day gets collected
using
[`journalctl`](https://www.freedesktop.org/software/systemd/man/latest/journalctl.html):

~~~
journalctl --identifier sshd --since=yesterday --until=today --priority=info --output=json
~~~

Next the IP addresses are extracted using [`jq`](https://jqlang.org/) or `jaq`:

~~~
jaq -r -c '
._SOURCE_REALTIME_TIMESTAMP as $ts 
| .MESSAGE 
| capture("from (?<ip>[0-9.]+) to") // capture("from \\[(?<ip>[^\\]]+)\\].* failed authentication") 
| {ts: $ts, ip: .ip}'
~~~

Finally the IP address is looked up using [OpenRDAP](https://github.com/openrdap/rdap).

~~~
location-lookup ip
~~~
