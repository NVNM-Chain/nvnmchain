[![License](https://img.shields.io/github/license/NVNM-Chain/nvnmchain)](https://github.com/NVNM-Chain/nvnmchain/blob/main/LICENSE)

# NVNMChain

NVNMChain is a blockchain platform for anchoring document hashes and off-chain artifacts on-chain, built as an ICS (Interchain Security) consumer chain. It uses the MANTRA EVM fork to expose EVM-compatible execution on top of Cosmos SDK, with custom modules for on-chain document anchoring and fee taxation.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Modules](#modules)
- [Getting Started](#getting-started)
- [Development](#development)
- [Security](#security)

## Overview

NVNMChain is an ICS opt-in consumer chain secured by a provider chain (e.g. MANTRAChain) via Interchain Security v7. It runs an EVM execution environment using a MANTRA fork of `cosmos/evm`, enabling Solidity smart contracts and EVM precompiles alongside native Cosmos SDK transactions.

The gas token is the IBC-wrapped MantraUSD token (`transfer/channel-1/erc20:<wMantraUSD address>`), bridged from the provider chain. As an ICS consumer chain, NVNMChain has no native staking at launch — validator security is inherited entirely from the provider chain.

## Architecture

NVNMChain is built on:

- **Cosmos SDK** (`MANTRA-Chain/cosmos-sdk v0.53.6`)
- **MANTRA EVM** (`MANTRA-Chain/evm v0.6.0-v8-mantra-1`) — EVM execution with ERC-20 module and feemarket
- **IBC Go v10** — cross-chain communication
- **Interchain Security v7** — validator set security from the provider chain
- **IBC Rate Limiting** — token transfer rate limits


## Modules

### `x/anchoring`

Manages registries and versioned records for anchoring off-chain artifacts (e.g. documents, certificates) on-chain. Each record contains a checksum, URI, optional metadata, and a status field. Records are versioned per checksum within a registry.

Key features:
- RBAC (role-based access control) scoped per registry or per checksum
- Roles: `admin`, `editor`
- EVM precompile at `0x0000000000000000000000000000000000000a00`

See [`x/anchoring/README.md`](x/anchoring/README.md) for full ABI, function selectors, and usage details.

### `x/tax`

Collects a configurable percentage of block fees and forwards them to a designated address. Runs in EndBlock before the CCV consumer module to ensure the tax cut is taken before any ICS reward distribution.

Parameters (governable):
- `tax.Tax` — fraction of fee collector balance to redirect (max 40%)
- `tax.TaxAddress` — destination address for collected tax

## Getting Started

### Prerequisites

- Go 1.24 or later

### Installation

```bash
git clone https://github.com/NVNM-Chain/nvnmchain.git
cd nvnmchain
make install
```

### Run a standalone single-node testnet

```bash
make build-and-run-single-node
```

This initialises a node with chain ID `test-chain`, native denom `anvnm`, and starts it locally.

### Run a full ICS consumer testnet (with provider + Hermes relayer)

From the `ccv-play` repository:

```bash
./local-testnet-opt-in-single.sh
```

The consumer JSON-RPC (EVM) endpoint will be available at `http://127.0.0.1:8541` with EVM chain ID `58886`.

## Development

### Build

```bash
make build
```

### Testing

#### Unit tests
```bash
make test-unit
```

#### E2E (interchain) tests

Build the Docker image and run:
```bash
make test-e2e
```

If the image is already built:
```bash
cd tests/interchain && go test -v -timeout 30m
```

### Linting

Uses `golangci-lint v2.12.1`. Run the same version as CI to avoid false positives.

#### Check
```bash
docker run -t --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.12.1 golangci-lint run
```

#### Fix
```bash
docker run -t --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.12.1 golangci-lint run --fix
```

### Generate protobuf

```bash
make proto-gen
```

### Generate docs

```bash
make docs
```

## Security

We take security seriously. If you discover a vulnerability, please follow the responsible disclosure process described in [SECURITY.md](SECURITY.md).

---

For module-level documentation, refer to the README files under the `x/` directory and architecture decisions under `adr/`.
