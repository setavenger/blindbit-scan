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
    "txid": "ce94df16dcfcc14a5bbee309bbd9b5f2eac7bf39a8b1f9fa54aa50fba541c27f",
    "vout": 0,
    "amount": 999500,
    "priv_key_tweak": "df942641627561f0888f031fb59044a9de8c2f0164d9103f86ca0dcee849d987",
    "pub_key": "f4460881dee3b13d15100f1a56ac973ba954941952bb40bd7626c3bf0814576c",
    "timestamp": 1720991230,
    "utxo_state": 2,
    "label": null
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

