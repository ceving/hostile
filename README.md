# Hostile attacks

This repository contains protocols of hostile attacks.

Once a day in the morning the data of the previous day gets collected
using
[`journalctl`](https://www.freedesktop.org/software/systemd/man/latest/journalctl.html):

	journalctl --identifier sshd --since=yesterday --until=today --priority=notice --output=json

Next the IP addresses are extracted using [`jq`](https://jqlang.org/):

	jq -c '{ts:._SOURCE_REALTIME_TIMESTAMP}+(.MESSAGE|capture("authentication failure.* rhost=(?<ip>[^ ]+) "))'

And the locations are looked up in the [IPFire
location](https://www.ipfire.org/location) database using
[`lookup-location`](https://codeberg.org/ceving/location-lookup).

	location-lookup ip
