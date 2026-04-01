# DIS PDU Registry — IEEE 1278.1-2012

## PDU Type Registry

| Type ID | Name | DIS Version | Description |
|---------|------|-------------|-------------|
| 0 | Other | 1 | Reserved |
| 1 | Entity State | 1 | Entity state update (ESP) |
| 2 | Fire | 1 | Weapon fire event |
| 3 | Detonation | 1 | Munition detonation/impact |
| 4 | Collision | 1 | Entity-entity physical collision |
| 5 | Service Request | 2 | Request for resupply |
| 6 | Resupply Offer | 2 | Offered resupply |
| 7 | Resupply Received | 2 | Resupply accepted |
| 8 | Resupply Cancel | 2 | Cancel resupply request |
| 9 | Repair Complete | 2 | Damage repair done |
| 10 | Repair Response | 2 | Repair action response |
| 11 | Create Entity | 2 | Request entity creation |
| 12 | Remove Entity | 2 | Request entity removal |
| 13 | Start/Resume | 2 | Resume federation |
| 14 | Stop/Freeze | 2 | Pause federation |
| 15 | Acknowledge | 2 | General acknowledgement |
| 16 | Action Request | 3 | Request action |
| 17 | Action Response | 3 | Action response |
| 18 | Data Query | 3 | Query data |
| 19 | Set Data | 3 | Set data |
| 20 | Data | 3 | Data response |
| 21 | Event Report | 3 | Event report |
| 22 | Comment | 3 | Comment/annotation |
| 23 | Electromagnetic Emission | 4 | IFF/SIGINT emission |
| 24 | Designator | 4 | Laser designator |
| 25 | Transmitter | 4 | Radio transmitter state |
| 26 | Signal | 4 | Modulated signal data |
| 27 | Receiver | 4 | Radio receiver state |
| 28 | IFF | 5 | Identification Friend/Foe |
| 29 | Underwater Acoustic | 5 | Sonar/ASW |
| 30 | Supplemental Emission/Entity State | 6 | SEES |
| 31 | Intercom Signal | 6 | Voice intercom |
| 32 | Intercom Control | 6 | Intercom control |
| 33 | Aggregate State | 6 | Formation/unit state |
| 34 | Is Group Of | 6 | Group membership |
| 35 | Transfer Ownership | 6 | Entity ownership transfer |
| 36 | Is Part Of | 6 | Aggregate membership |
| 37 | Minefield State | 6 | Minefield state |
| 38 | Minefield Query | 6 | Query minefield |
| 39 | Minefield Data | 6 | Minefield data response |
| 40 | Minefield Response NACK | 6 | Negative ack |
| 41 | Environmental Process | 7 | Environmental update |
| 42 | Gridded Data | 7 | Weather/ocean grid |
| 43 | Point Object State | 7 | Point object |
| 44 | Linear Object State | 7 | Linear object |
| 45 | Areal Object State | 7 | Areal object |
| 46 | TSPI | 7 | Time Space Position (Live Entity) |
| 47 | LE Appearance | 7 | Live Entity appearance |
| 48 | LE Articulated Parts | 7 | LE articulated parts |
| 49 | LE Fire | 7 | Live Entity fire |
| 50 | LE Detonation | 7 | Live Entity detonation |
| 51 | Create Entity R | 7 | With reasoning |
| 52 | Remove Entity R | 7 | With reasoning |

## Entity Type Codes (Platform Kinds)

| Kind | Domain |
|------|--------|
| 0 | Other |
| 1 | Platform (tank, aircraft, ship) |
| 2 | Munition (bullet, missile, bomb) |
| 3 | Lifeform (infantry) |
| 4 | Environmental |
| 5 | Cultural Feature |
| 6 | Supply (ammo, fuel, food) |
| 7 | Radio |
| 8 | Projectile |
| 9 | Explosion |
| 10 | Information |
