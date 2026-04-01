# HLA/DIS/TENA Gateway — DIS-HLA Bridge

## Overview

The HLA Bridge (`hla-bridge`) translates between DIS PDUs and HLA interactions, enabling interoperability between DIS-based entity simulations and HLA federations.

## Architecture

```
DIS Network (IEEE 1278.1)          HLA RTI (IEEE 1516)
    │                                   │
    │  DIS Entity State PDU ────────────►│  ObjectUpdate (Entity)
    │  DIS Fire PDU ────────────────────►│  Interaction (Fire)
    │  DIS Detonation PDU ──────────────►│  Interaction (Detonation)
    │                                    │
    │◄── HLA ObjectUpdate ───────────────│  Entity State
    │◄── HLA Interaction ───────────────│  Fire/Detonation
```

## Supported Mappings

| DIS PDU | HLA Interaction/Object |
|---------|----------------------|
| Entity State (1) | Update Attribute (Entity) |
| Fire (2) | WeaponFire Interaction |
| Detonation (3) | Detonation Interaction |
| Collision (4) | Collision Interaction |
| Action Request (16) | Action Request Interaction |
| Data Query (18) | Data Request Interaction |

## Build

```bash
go build -o hla-bridge .
```

## Run

```bash
./hla-bridge \
  --dis-multicast 239.255.0.1 \
  --dis-port 3000 \
  --hla-host localhost \
  --hla-port 40000 \
  --federation-name TROOPER-FORGE
```

## DIS PDU → HLA Mapping Rules

1. Entity State PDU → HLA Entity Object (Update Attributes)
   - Entity ID → Object ID
   - Location/Orientation/Velocity → Spatial attributes
   - Force ID → Force affiliation
   - Entity Type → Entity type record

2. Fire PDU → HLA WeaponFire Interaction
   - Fire Event ID + Munition + Location
   - Fire Entity ID (firing unit)
   - Target Entity ID (if available)

3. Detonation PDU → HLA Detonation Interaction
   - Detonation Event ID + Location
   - Munition type + warhead
   - Firing + Target Entity IDs
