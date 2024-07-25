# blindbit-scan

A scan-only client for BIP 352 silent payment wallets. The program runs in the background and connects to a blindbit-oracle instance (and optionally an electrum server). Found UTXOs can be retrieved via a simple HTTP endpoint.


## Setup

### Build
- clone this repo
- have go 1.22.4+ installed
- run `make build` in project root directory 

### Configuration
The necessary fields are explained in the [blindbit.example.toml](./blindbit.example.toml).

### Run
- setup the blindbit.toml with the correct configurations
- NOTE: [devtools](https://github.com/setavenger/blindbitd/tree/master/devtools) from [blindbitd](https://github.com/setavenger/blindbitd) can help get the scanonly keys for blindbit-scan
- run the binary pointing to the datadirectory containing the blindbit.toml
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

