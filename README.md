# ha-adapters for MQTT

Initially influenced by [https://github.com/dchesterton/amcrest2mqtt/](https://github.com/dchesterton/amcrest2mqtt/), this project is to promote some
better type-safe error handling for longevity and reliability of the integration.  Combines key functionality across amcrest2mqtt and python
supporting library.

Generally written to be extensible to other devices that may want to publish sensors to MQTT.

## AD410 Doorbell MQTT Adapter

The `AD410` adapter integrates with the Amcrest Doorbell AD410.  To use, you need a small set of either environment
or CLI variables:

```sh
AD410_URL=http://doorbell-hostname
AD410_USERNAME=admin #almost always 'admin'
AD410_PASSWORD=<password from app>
MQTT_URI=<MQTT URI>
#optionally:
MQTT_USERNAME=
MQTT_PASSWORD=
```

For example, to run as a docker container:
```sh
docker run -d \
    -e AD410_URL=http://doorbell-hostname \
    -e AD410_USERNAME=admin \
    -e AD410_PASSWORD=password \
    -e MQTT_URI=hostname:1883 \
    zix99/ha-ad410:latest
```

No persistent volumes necessary

# License

    Copyright (C) 2023  Christopher LaPointe

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
