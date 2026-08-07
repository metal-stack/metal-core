# metal-core

metal-core dynamically reconfigures switches based on the state held in the metal-api. Therefore, it must run on every leaf switch and have control over the configuration files for network interfaces and the routing suite (`/etc/frr/frr.config`) of the switches.

In the PXE-boot process of machines `metal-core` will act as a proxy between API-requests issued by `pixiecore` and the `metal-api`. The `metal-api` will answer with a mini OS (see [metal-hammer](https://github.com/metal-stack/metal-hammer) and [kernel](https://github.com/metal-stack/kernel)).

Besides that, it ensures the proper boot order (IPMI) and monitors their liveliness with [LLDP](https://github.com/metal-stack/go-lldpd)).

## Build

Ensure you have `libpcap-dev` installed.

```bash
make
```

## Interface Naming on SONiC Switches

On SONiC switches, there are different naming schemas for interfaces.
For example, the first port could be named `Ethernet0`, or it could be `Eth1/1` or something similar.
Additionally, a port can have an alias, which may be the same as its name or it may follow a different naming schema.
If the port is named `Ethernet0` the alias could be `Eth1/1(Port1)`.
But it could also be the other way round.
The defaults for the naming schema differs across distributions.

These differences wouldn't cause any problems if it weren't for LLDP.
An LLDP message carries two fields to identify the port, `portidsubtype` and `portdescription`.
Depending on the distribution `portidsubtype` will be either the port's name or its alias and `portdescription` will be the other of the two.
So the metal-core registers its ports at the metal-api which stores the names and aliases as `Nic.Name` and `Nic.Identifier`.
At the same time, when a machine registers at the metal-api it reports its LLDP neighbors and identifies the neighbors' ports by `portidsubtype` and `portdescription`.
When the metal-api attempts to match the machine's neighbors with the switches' ports it compares the neighbor's `portidsubtype` with all of the switch's Nics' `.Identifier` field.
But this only works if the port's alias is identical to its `portidsubtype` which, as stated above, is not always the case.

While it is possible to configure `portidsubtype` and `portdescription` via `lldpcli`, this configuration is not persisted.
A simple change of the MTU will restore the defaults.
So this is not a viable solution.

To accommodate all of the possible combinations, there is an `InterfaceNamingSchema` option.
This option has nothing to do with SONiC's `interface-naming`.
It simply tells metal-core how it should report its ports to the metal-api.
It allows the following values.

- `default`: `V1SwitchNic.Name` is the interface name; `V1SwitchNic.Identifier` is the interface alias
- `swap`: name and alias get swapped
- `name`: both `V1SwitchNic.Name` and `V1SwitchNic.Identifier` will be the interface name
- `alias`: both `V1SwitchNic.Name` and `V1SwitchNic.Identifier` will be the interface alias

To determine which of these suits your setup compare the port aliases with the LLDP configuration for `portidsubtype`.
If they match the correct value for `InterfaceNamingSchema` is `default`.
If not, check if `portdescription` matches the alias and `portidsubtype` matches the name.
In that case you can use `swap`.
If you need to have both of the fields to have the same value, use either `name` or `alias`.
