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
The defaults for the naming schemas differ across distributions.

These differences wouldn't cause any problems if it weren't for LLDP.
An LLDP message carries two fields to identify the port, `portidsubtype` and `portdescription`.
Depending on the distribution `portidsubtype` will be either the port's name or its alias and `portdescription`, usually, will be the other of the two.
So the metal-core registers its ports at the metal-api which stores the names as `V1SwitchNic.Name` and aliases as `V1SwitchNic.Identifier`.
At the same time, when a machine registers at the metal-api it reports its LLDP neighbors and identifies the neighbors' ports by `portidsubtype` and `portdescription`.
When the metal-api attempts to match the machine's neighbors with the neighboring switches' ports it compares each neighboring switch's `portidsubtype` with all of its Nics' `.Identifier` fields.
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

## Layer 3 Boot Mode (MEP-20)

By default (`METAL_CORE_BOOT_MODE=pxe`) the ports of unprovisioned machines are put into the PXE VLAN (`METAL_CORE_PXE_VLAN_ID`) and machines boot via DHCP/PXE.

With `METAL_CORE_BOOT_MODE=l3` (SONiC only) metal-core instead puts every unprovisioned port into the IPv6 boot VRF `VrfBoot` as described in [MEP-20](https://metal-stack.io/community/MEP-20-full-layer-3-dataplane). This requires a **boot network** in the partition of the switch: a network of type `boot` in the metal-apiserver with an IPv6 prefix, a `vrf` (the layer 3 VNI of `VrfBoot`) and a default child prefix length of 64 for IPv6, for example:

```bash
metalctlv2 admin network create --id boot-mini-lab --name "Boot Network" --type boot --partition mini-lab \
  --prefixes fd00:20::/48 --vrf 104000 --default-ipv6-prefix-length 64 --nat-type none
```

- The metal-apiserver assigns every switch port its own `/64` from the boot network when the switch registers (or when the boot network is created after the switch registered) and delivers it as `boot_prefix` of the switch nic, together with the `boot_vni` of the switch. metal-core takes both from there; nothing about addresses is configured on the switch itself.
- `VrfBoot` is created with the boot VNI and mapped to a switch-local VLAN and VXLAN tunnel like a tenant VRF, so the boot prefixes are reachable across the EVPN fabric.
- Every unprovisioned port becomes a routed interface in `VrfBoot` with its boot prefix, the switch takes the first address of the prefix. Ports without an assigned boot prefix are skipped with a warning until the metal-apiserver has assigned one.
- FRR sends router advertisements with the prefix on every port, so machines configure their address via SLAAC. The IPv6 DNS servers of the partition's boot configuration (`rdnss`, delivered as `boot_rdnss` of the switch) are advertised as RDNSS option.
- `VrfBoot` gets a BGP instance that redistributes the connected prefixes into EVPN. It has no BGP neighbors.
- All ACL tables in `CONFIG_DB` whose name starts with `BOOT_` are bound to the unprovisioned ports. The tables and rules are static and are generated by [sonic-configdb-utils](https://github.com/metal-stack/sonic-configdb-utils) (`boot_acl`). metal-core only maintains the port binding and refuses to run in l3 mode if no such table exists.

Ports of allocated machines and firewalls are moved out of `VrfBoot` (their addresses and the ACL binding are removed) and back when the machine is freed. Migration from PXE to ISO/L3 boot is one-way; reverting a partition to PXE is unsupported.

| Environment variable | Default | Description |
|---|---|---|
| `METAL_CORE_BOOT_MODE` | `pxe` | `pxe` or `l3` |

## API boundary test

With matching `metal-apiserver` and `metal-core` checkouts, run from the
`metal-apiserver` directory:

```sh
make test-metal-core-boundary METAL_CORE_DIR=../metal-core
```

This opt-in test starts the apiserver's end-to-end harness and runs metal-core's
registration and polling loop in a separate Go test process over HTTP. It checks
infra-token permissions, per-port boot prefixes, boot VNI/RDNSS, heartbeat
persistence, re-registration, a partition RDNSS update, and reporting/recovery
after a backend apply failure. Both test processes run with the race detector.

The NOS and local network observations are controlled fixtures. The test does
not configure host interfaces or validate SONiC/FRR or hardware ACL enforcement.
Docker is required for the apiserver test dependencies. Both checkouts must use
an API version containing the MEP-20 fields, either through a shared `go.work`
or released dependencies. The `boundary` build tag excludes this test from the
ordinary unit-test suite; invoking it without the harness fails explicitly.
