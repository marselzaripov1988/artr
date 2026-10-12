# Artery Network: Cosmos SDK 0.42 → 0.53 migration

[![CI](https://github.com/marselzaripov1988/artr/actions/workflows/ci.yml/badge.svg?branch=feat/ibc)](https://github.com/marselzaripov1988/artr/actions/workflows/ci.yml)
![Cosmos SDK](https://img.shields.io/badge/Cosmos%20SDK-v0.53.8-blue)
![CometBFT](https://img.shields.io/badge/CometBFT-v0.38-blue)
![IBC](https://img.shields.io/badge/ibc--go-v10-blue)

[Artery Network](https://artery.network/) is a live Cosmos SDK chain with
eight custom modules and about 383,000 accounts on mainnet. Its node was
stuck on **Cosmos SDK v0.42.6 / Tendermint v0.34 / Go 1.15**. That stack has
been end-of-life for years and gets no security patches.

This fork moves the chain to the **2025.1 release family (SDK v0.53.8,
CometBFT v0.38, ABCI 2.0)**, one major at a time, and then connects it to
IBC. The work was done by [Marsel Zaripov](https://github.com/marselzaripov1988).
Upstream: [arterynetwork/artr](https://github.com/arterynetwork/artr).

> **Status:** every hop builds and passes the module test suites. The 0.53
> build runs on a devnet/testnet stand, where transactions, governance and a
> full coordinated-halt rehearsal have been checked. The mainnet switch-over
> is the planned next step and has **not** happened yet.

## One hop per branch

Each hop is its own branch, so every SDK upgrade can be reviewed on its own.

| Branch | SDK | Consensus | Go | What the hop required |
|---|---|---|---|---|
| `master` | v0.42.6 | Tendermint v0.34.11 | 1.15 | upstream starting point |
| [`migrate/sdk-0.45`](https://github.com/marselzaripov1988/artr/compare/master...migrate/sdk-0.45) | v0.45.16 | Tendermint v0.34.27 | 1.19 | codec renames, `ConsensusVersion` for all modules, `SupplyI` / `sdk.AddrLen` removed, legacy bech32 validator keys |
| [`migrate/sdk-0.46`](https://github.com/marselzaripov1988/artr/compare/migrate/sdk-0.45...migrate/sdk-0.46) | v0.46.16 | Tendermint v0.34.29 | 1.19 | legacy REST removed, store-key types, variadic `Coins.Sub`, code-level **Tendermint → CometBFT** rename |
| [`migrate/sdk-0.47`](https://github.com/marselzaripov1988/artr/compare/migrate/sdk-0.46...migrate/sdk-0.47) | v0.47.17 | CometBFT v0.37 | 1.19 | legacy router/querier removed, **params moved out of `x/params` into each of the 8 modules' stores**, `x/params` dropped |
| [`migrate/sdk-0.50`](https://github.com/marselzaripov1988/artr/compare/migrate/sdk-0.47...migrate/sdk-0.50) | v0.50.15 | CometBFT v0.38 | 1.22 | `cosmossdk.io/*` package split, `appmodule` interfaces, **ABCI 2.0 / `FinalizeBlock`**, three-state vote flag in validator liveness |
| [`migrate/sdk-0.53`](https://github.com/marselzaripov1988/artr/compare/migrate/sdk-0.50...migrate/sdk-0.53) | v0.53.8 | CometBFT v0.38 | 1.23 | only a version bump: 0.50 had already done the hard part |
| [`feat/ibc`](https://github.com/marselzaripov1988/artr/compare/migrate/sdk-0.53...feat/ibc) **(default)** | v0.53.8 | CometBFT v0.38 | 1.23 | IBC core and ICS-20 transfer (ibc-go v10), slashing for double-signing, PreBlock / upgrade fixes, CI |

**Why stop at 0.53 and not 0.55?** 0.53 is the oldest release family that
still gets security patches, and it keeps CometBFT v0.38. The jump to 0.55
also brings CometBFT v0.40, which is another consensus-layer change. That
hop is safer to start from a base that is already proven on mainnet.

Scope: 56 commits, about 150 files changed in `app/`, `cmd/`, `x/` and
`util/` (generated code and vendored files not counted).

## Bugs the migration surfaced

Most of these were silent: nothing logged an error, the chain kept running
and the damage showed up later. Each fix commit explains the cause in detail.

- **Upgrades scheduled but never executed** ([bea8005](https://github.com/marselzaripov1988/artr/commit/bea8005)).
  Since 0.50, `x/upgrade` checks the plan in `PreBlock`, but the app never
  called it. Governance passed, the plan was stored, the halt height went by
  and the chain kept going. I caught it by rehearsing the full cycle on the
  stand. After the fix, both nodes halted at the planned height.
- **Upgrade proposals rejected by the SDK** ([612cb9a](https://github.com/marselzaripov1988/artr/commit/612cb9a)).
  Artery scheduled upgrades by time. Since 0.47, `Plan.ValidateBasic` rejects
  time-based plans. Plans are now scheduled by height, and the old time-based
  proposals in the chain's history still validate.
- **No transaction could be sent** ([90e78cb](https://github.com/marselzaripov1988/artr/commit/90e78cb)).
  Since 0.50, the signer is read from the `cosmos.msg.v1.signer` proto
  option, not from `GetSigners()`. I annotated all 28 message types in 7
  modules and checked each against the old `GetSigners()`. The first
  transaction on the testnet in 91,985 blocks went through right after.
- **Every state query failing because of one empty store** ([d5f0ccd](https://github.com/marselzaripov1988/artr/commit/d5f0ccd)).
  Once the params had moved out, the empty `x/params` tree had no version.
  `CacheMultiStoreWithVersion` then failed every query, whatever module it
  was for.
- **Writes and reads using different key prefixes** ([71e66ea](https://github.com/marselzaripov1988/artr/commit/71e66ea)).
  After the params moved, some keeper paths still used the raw store. The
  validator set came back empty and the node crashed on start.
- **Genesis export reading past the end of a slice** ([c9f3308](https://github.com/marselzaripov1988/artr/commit/c9f3308)).
  The `sdk.AddrLen` replacement used the 32-byte ADR-028 length, but Artery
  addresses are 20 bytes. The bug went unnoticed through three hops because
  devnets never export state.
- **Latent IBC init failure** ([184d71e](https://github.com/marselzaripov1988/artr/commit/184d71e)).
  `InitGenesis` only runs for modules that have a section in genesis, and the
  mainnet export has no `ibc` section. The chain would run normally and then
  fail on its first state export.
- **Consensus-key format kept byte-for-byte** ([7993ee5](https://github.com/marselzaripov1988/artr/commit/7993ee5)).
  `legacybech32` is gone in 0.50, but validator keys in genesis, in state and
  in messages depend on it. I re-implemented the encoding and checked it
  against the original on 100 random keys while the original still existed.

## Building and testing

Requires Go 1.23+. `go.mod` pins the `go1.24.13` toolchain, and the Go
command downloads it automatically.

```sh
go build ./cmd/artrd
go test -tags testing ./...        # 18 packages
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) builds from a
clean clone, runs the tests and regenerates protobuf code in Docker. The run
fails if the generated code differs from what is committed.

Instructions for running an upstream node (v2.5.x) are kept in
[docs/upstream-README.md](docs/upstream-README.md).

