# BlindBit Scan

A scan-only client for BIP 352 silent payment wallets. The program runs in the background and connects to a blindbit-oracle instance (and optionally an electrum server). Found UTXOs can be retrieved via a simple HTTP endpoint.


## Setup

### Build
- clone this repo
- have go 1.22.4+ installed
- run `make build` in project root directory 

### Configuration
The necessary fields are explained in the [blindbit.example.toml](./blindbit.example.toml). Alternatively the most variables can be set as ENV variables. 
A simple frontend exists as well. [The frontend for BlindBit Scan](https://github.com/setavenger/blindbit-scan-frontend) has functionality to set up the keys and also download a json file with the UTXOs.

### Run
- setup the blindbit.toml with the correct configurations
- NOTE: [devtools](https://github.com/setavenger/blindbitd/tree/master/devtools) from [blindbitd](https://github.com/setavenger/blindbitd) can help get the scanonly keys for blindbit-scan
- run the binary pointing to the data directory containing the blindbit.toml
```bash
blindbit-scan --datadir /directory/containing/configfile
```

## Endpoints

`/utxos` - returns a json array of UTXOs which have been found.
```json
[
  {
    "txid": "66cf6460207e957ff77b1cad191050a8623d36671e94a46813b4bc10e6b35b6c",
    "vout": 0,
    "amount": 12000000,
    "priv_key_tweak": "6ffdbe4ab9c40a43edf31394ba8226475e610cd1a0d5da808248c1c9d6d79056",
    "pub_key": "bea89f2f17a7f438f4d5ab495d9a68a5d8ed3c7b5166f7427a6c39e6d9e3b062",
    "timestamp": 1721944866,
    "utxo_state": "spent",
    "label": null
  },
  {
    "txid": "66cf6460207e957ff77b1cad191050a8623d36671e94a46813b4bc10e6b35b6c",
    "vout": 1,
    "amount": 55990460,
    "priv_key_tweak": "ce2be4d8974a0852ac6c791c8d6332a1044d1b5224dc4da36522e9ff49150a07",
    "pub_key": "9326bdcdd477bf09d4fd3e39af62d9b3b0e0526c02d66b7ad4e0f80430cc1527",
    "timestamp": 1721944866,
    "utxo_state": "unspent",
    "label": {
      "pub_key": "02504188df0e7d4c1559e8d7e1d4c4c417086824ff37ddd98afbcc3a461430f1bd",
      "tweak": "cea33be68bbdeed59859bbc3b3afc8798564d3afc2690d68bb7223cb0d481dc0",
      "address": "tsp1qqt7u5h5n4cw8yctkednnnydytcuwmhz5xkdv0qtmscx90dwu06s5yq62ft33x5a2c605knje7u7c6fmfjvmjkq5xpchzr5xlqzguhwcyfc8gw326",
      "m": 0
    }
  }
]
```

`/height` - returns the last height the program has scanned.
```json
{
  "height": 204472
}
```

`/address` - returns the default/non-labeled address for the currently set keys.
```json
{
  "address": "tsp1qqt7u5h5n4cw8yctkednnnydytcuwmhz5xkdv0qtmscx90dwu06s5yq4h68jgkn3qukqrw8mcgmt6k8lytvpzfd49xjmtdjuq24yffypltux9f9f9"
}
```

`/new-nwc-connection` - creates a new NWC connection string
```json
{
  "uri": "nostr+walletconnect://28c1d46a01f54ed3a344b906a92fa1947b53be85d880ccfef292cced35cf33cc?relay=wss://relay.getalby.com/v1&secret=bea5e03730764f0d70fb5b28939cd6e03c3c33323b97aa89971991f328b9da43"
}
```

## Nostr Wallet Connect
In addition to the standard UTXO endpoints BlindBit Scan allows for a NWC style
communication between clients and this server. The user can call
`new-nwc-connection` and use the received connection string in [Blindbit
Spend](https://github.com/setavenger/blindbit-spend) or in the PWA app
[BlindBit-PWA](https://github.com/setavenger/blindbit-silentium). The two
methods supported are `get_info` and `list_utxos`. `get_info` has pretty much
same format as the standard Nostr Wallet Connect spec. `list_utxos` has the
same output as the endpoint `/utxos` just in the NWC format. Please open an
issue if you find something not working properly.

## Support me
I'm building and maintaining the BlindBit suite in my free time. I'm grateful
for any contributions, be it feedback, PRs or hard sats.

LNURL: `setor@getalby.com`
SP-Address: `sp1qqwgst7mthsl46hkcek6ets58rfunr4qaxpchuegs09m6uy3tm4xysqmdf6xr9rh68stzzhshjt6z7288tc7eqts65ls4sg2dg6aexlx595f5wa7u`
