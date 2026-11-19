# metal-core

metal-core dynamically reconfigures switches based on the state held in the metal-api. Therefore, it must run on every leaf switch and have control over the configuration files for network interfaces and the routing suite (`/etc/frr/frr.config`) of the switches.

In the PXE-boot process of machines `metal-core` will act as a proxy between API-requests issued by `pixiecore` and the `metal-api`. The `metal-api` will answer with a mini OS (see [metal-hammer](https://github.com/metal-stack/metal-hammer) and [kernel](https://github.com/metal-stack/kernel)).

Besides that, it ensures the proper boot order (IPMI) and monitors their liveliness with [LLDP](https://github.com/metal-stack/go-lldpd)).

## Build

Ensure you have `libpcap-dev` installed.

```bash
make
```

## Static BGP Config

Typically, metal-core configures the ports based on information is receives from the API.
Ports that are connected to unprovisioned machines will be added to the PXE VLAN.
Once a machine is allocated the port it is connected to is bound to the tenant VRF of the machine.

Additionally, you may add some custom BGP configuration by passing a yaml file via the `METAL_CORE_STATIC_BGP_CONFIG_FILE` env var.
The yaml file takes the following fields:

```yaml
# VRFs will create a L3VNI where the VNI is the same number as the VRF number.
vrfs:
  # VRF name must be of the form Vrf<VNI> (on SONiC) or vrf<VNI> (on Cumulus).
  - Vrf100

# For each unnumbered interface `ipv6 nd ra-interval 6` and `no ipv6 nd suppress-ra` will be configured
# Each interface that you want to establish a BGP session on must be included here with the correct VRF.
unnumberedinterfaces:
  # Name of the interface
  - name: Ethernet0

    # VRF of the interface.
    # Omit for default VRF.
    vrf: Vrf100

# All BGP session configurations, including peer-groups, neighbors and address-families.
bgps:
  # The VRF for which the BGP is configured.
  # Omit for default VRF.
  - vrf: Vrf100

    # A list of peer-groups to configure.
    peergroups:
      # Name of the peer-group.
      - name: MACHINE

        # Commands to configure the peer-group.
        # The instantiating command is added by default, no need to add it here.
        commands:
          - remote-as external
          - timers 2 8
          - ebgp-multihop 2

    # A list of neighbors.
    neighbors:
      # Name of the neighbor.
      # An unnumbered neighbor must include the `interface` suffix.
      - name: Ethernet0 interface
        # The neighbor's peer-group.
        # May be omitted.
        peergroup: MACHINE

      # Example for a numbered session.
      - name: 10.0.0.1
        peergroup: MACHINE

    # Configuration for the address-families `ipv4 unicast`, `ipv6 unicast` and `l2vpn evpn`.
    addressfamilies:
      # address-family ipv4 unicast
      ipv4unicast:
        # Global commands for this address-family.
        globalcommands:
          - redistribute connected route-map LOOPBACKS

        # Neighbor-specific configuration.
        neighbors:
          # Name of a neighbor by interface (without the `interface` suffix), by IP address, or by peer-group
          - name: MACHINE
            # Neighbor-specific commands
            commands:
              - maximum-prefix 24000
              - route-map Vrf100-in in

      # same configuration options as for ipv4unicast
      ipv6unicast:
        # ...

      # same configuration options as for the other address-families.
      l2vpnevpn:
        # ...

# List of route-maps.
routemaps:
  # Name of the route-map.
  - name: Vrf100-in
    # List of entries for this route-map.
    entries:
      - match ip address prefix-list Vrf100-in-prefixes
    # Policy may be `deny` or `permit`.
    policy: permit
    # Order must allow for a total order of the route-maps of the same name, i.e. there cannot be two route-maps with the same name and same order.
    order: 10

# List of prefix-lists.
prefixlists:
  # Address family is `ip` for IPv4 and `ipv6` for IPv6.
  - addressfamily: ip
    # Name of the prefix-list.
    name: Vrf100-in-prefixes
    # Spec defines what prefixes to include in the prefix-list.
    # The prefix must match the address family.
    spec: permit 10.1.1.0/24 le 32
```

Static BGP configuration can be used to connect external hosts, i.e. hosts that are not registered machines, to a network that is managed by our API.
Or you can add additional route-maps and prefix-lists to the default config, e.g. if you need to filter advertisements in the underlay more strictly than the metal-core does by default.

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
