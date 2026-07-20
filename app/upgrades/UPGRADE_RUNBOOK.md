# v1.2.0 / v1.3.0 upgrade runbook (operator instructions)

The mainnet case-law export (`mainnet-full-export/`: **2,114 registries, 11,944,960 records**,
3.6 GB uncompressed / ~458 MB gzipped) is loaded in **two separate chain upgrades** rather
than one, to keep each upgrade's peak memory footprint manageable:

| Upgrade | Tranches | Registries | Records |
|---|---|---:|---:|
| **v1.2.0** | 1–3 (federal appellate, federal complete, state pilot) | 886 | 5,191,603 |
| **v1.3.0** | 4 (state remainder) | 1,228 | 6,753,357 |

Both upgrades share the same mechanism: the bulk tranche data is **not** embedded in the
binary — every validator must independently stage it on local disk before each upgrade
height, and each upgrade handler verifies its slice of the export byte-for-byte against
checksums baked into the binary before writing anything to state. Repeat this whole runbook
for v1.3.0 at its own upgrade height — staging and verification are not shared between the
two upgrades, only the underlying export data is.

Read this whole document before starting either upgrade. Steps 1–2 (for whichever upgrade is
next) must be done **before** that upgrade's height is reached; there is no way to supply
this data after the fact.

## What's baked into the binary (nothing to do here)

- Each upgrade embeds its own `registries.json`/`manifest.json` slice (v1.2.0:
  `app/upgrades/v1_2/data/`, tranches 1–3; v1.3.0: `app/upgrades/v1_3/data/`, tranche 4 only) —
  compiled into the `nvnmchaind` binary from source. You don't need to source these
  separately; they're identical across every validator that built from the same release.
- The registry admin/creator address (`nvnm14a3em3mr9mvta9ccgk80wn0dxgzt5lkt2r8trx`) is also
  hardcoded in the binary for both upgrades. This is set once per registry and can never be
  changed by any message handler, so there is nothing for you to configure here either.

## What you must do (once per upgrade)

### 1. Obtain the export and stage it locally

Get the `mainnet-full-export/` bundle via the agreed handoff channel (it is not distributed
via this repo — it's ~460 MB and gitignored). It must contain, unmodified:

```
mainnet-full-export/
├── registries.json
├── manifest.json
├── tranche-1-federal-appellate/*.jsonl.gz
├── tranche-2-federal-complete/*.jsonl.gz
├── tranche-3-state-pilot/*.jsonl.gz
└── tranche-4-state-remainder/*.jsonl.gz
```

It's the same bundle for both upgrades — each upgrade's manifest only reads its own
tranche(s) and ignores the rest, so you don't need a trimmed copy per upgrade.

Copy the **entire directory as-is** to this exact path under your node's home directory,
using the subfolder matching the upgrade you're about to run:

```
<node-home>/upgrades/v1_2/mainnet-full-export/   # before the v1.2.0 upgrade height
<node-home>/upgrades/v1_3/mainnet-full-export/   # before the v1.3.0 upgrade height
```

For example, if your node home is `~/.nvnmchain`, staging for v1.2.0:

```bash
mkdir -p ~/.nvnmchain/upgrades/v1_2
cp -R /path/to/mainnet-full-export ~/.nvnmchain/upgrades/v1_2/mainnet-full-export
```

...and later, staging the same bundle again for v1.3.0:

```bash
mkdir -p ~/.nvnmchain/upgrades/v1_3
cp -R /path/to/mainnet-full-export ~/.nvnmchain/upgrades/v1_3/mainnet-full-export
```

If you'd rather not keep two full copies on disk, stage it once and point both upgrades at
the same location via the env var override instead (see below) — a symlink works too.

If you need to stage it somewhere other than under the node home (different disk, shared
mount, etc.), set the matching environment variable on the node process instead — it
overrides that upgrade's default path entirely:

```bash
export NVNMCHAIN_V1_2_EXPORT_DIR=/mnt/some-other-path/mainnet-full-export   # for v1.2.0
export NVNMCHAIN_V1_3_EXPORT_DIR=/mnt/some-other-path/mainnet-full-export   # for v1.3.0
```

### 2. Verify the staged data yourself before the upgrade height

Each upgrade handler *will* verify every file it needs against `manifest.json` and refuse to
proceed on any mismatch — but finding that out at the upgrade height means the chain is
already stalled waiting on you. Verify locally first so a bad copy is caught with time to fix
it. From inside the staged `mainnet-full-export/` directory:

```bash
python3 - <<'EOF'
import json, hashlib, pathlib, sys

base = pathlib.Path(".")
manifest = json.loads((base / "manifest.json").read_text())

bad = []
for f in manifest["files"]:
    path = base / f["file"]
    if not path.exists():
        bad.append((f["file"], "missing"))
        continue
    got = hashlib.sha256(path.read_bytes()).hexdigest()
    if got != f["sha256_gz"]:
        bad.append((f["file"], f"sha256 mismatch: want {f['sha256_gz']}, got {got}"))

print(f"checked {len(manifest['files'])} files, {len(bad)} problems")
for name, reason in bad:
    print(f"  {name}: {reason}")
sys.exit(1 if bad else 0)
EOF
```

This checks the full bundle's manifest (all 4 tranches) — harmless to run once per staged
copy even though a given upgrade only reads a subset of it. Do not proceed to the upgrade
height until this reports zero problems.

### 3. Confirm disk headroom and free RAM

Every write inside an upgrade handler is held in an **in-memory cache** until the entire
block commits at the very end — nothing hits disk until it's completely done. This means:

- Peak RAM usage during the upgrade scales with that upgrade's record count (v1.3.0, at
  6.75M records, will use noticeably more memory than v1.2.0's 5.19M). On a memory-constrained
  machine this can be severe enough to risk an OOM kill — confirm you have several GB of
  headroom free (not just "not currently swapping") before starting either upgrade, and close
  other memory-heavy processes if you're tight.
- Only after the handler finishes does the real IAVL tree write + disk commit happen — this
  is a separate, silent phase (no additional log lines) that can itself take a while. Watch
  `<node-home>/data/application.db` size to see when this phase actually starts.
- Writing this data into the IAVL store will grow your node's `data/` directory by more than
  the raw uncompressed size (tree node overhead, historical versions, indexing). Confirm
  comfortable free disk space beyond the raw record data size — err generous; this has not
  been precisely benchmarked yet.

If either upgrade handler fails partway (including via an OOM kill), nothing has committed —
see "If it fails" below. It's safe to fix the environment and retry.

### 4. Run the upgrade as normal, then watch for these log lines

**v1.2.0** — tranches 1–3:

```
Starting v1.2.0 upgrade...
Running module migrations...
Seeding 886 anchoring registries...
Seeded 886 anchoring registries
Loading tranche 1 records...
Loading tranche 2 records...
Loading tranche 3 records...
Seeded 5191603 anchoring records
Upgrade v1.2.0 complete
```

**v1.3.0** — tranche 4, run later at its own upgrade height:

```
Starting v1.3.0 upgrade...
Running module migrations...
Seeding 1228 anchoring registries...
Seeded 1228 anchoring registries
Loading tranche 4 records...
Seeded 6753357 anchoring records
Upgrade v1.3.0 complete
```

**Expect an extended pause in block production while either runs** — these are millions of
sequential state writes executed synchronously inside a single upgrade block each, not spread
across multiple blocks. There is no published ETA yet for either; if you have the opportunity
to rehearse against a copy of chain state (or an equivalent-size testnet) beforehand, do it
and share the observed duration with other operators so everyone knows what to expect before
the real cutover.

Do not restart or kill the node while this is running just because it looks stalled — check
for forward progress (new "Loading tranche N..." lines, climbing RSS in `ps`, or eventual
growth in `application.db` once the handler finishes) before concluding something is wrong.

## If it fails

Both handlers are fail-closed: any sha256 mismatch, missing file, or record/registry count
mismatch against `manifest.json` aborts the upgrade with a clear error naming the offending
file and expected-vs-actual value, instead of loading partial or incorrect data. Because this
happens before the block commits, **no partial state is persisted** — an OOM kill has the same
effect. Fix the staged data or free up memory (re-copy, re-verify per step 2), and restart the
node; the upgrade will retry cleanly from the same height.

Common causes:
- Wrong path — double check the upgrade-specific path (`.../upgrades/v1_2/mainnet-full-export/`
  or `.../upgrades/v1_3/mainnet-full-export/`, or your `NVNMCHAIN_V1_2_EXPORT_DIR` /
  `NVNMCHAIN_V1_3_EXPORT_DIR` override) actually contains the tranche directories, not a
  nested copy of `mainnet-full-export/mainnet-full-export/`.
- Incomplete transfer/copy — re-run step 2's verification.
- Wrong export version — confirm you were handed the export matching this binary's release
  (its `manifest.json` totals should read 2,114 registries / 11,944,960 records across all 4
  tranches).
- Insufficient memory — see step 3.

## Post-upgrade verification

After v1.2.0:

```bash
nvnmchaind query anchoring registries --output json | jq '.registries | length'
# expect: 886

nvnmchaind query anchoring registry <registry-id> --output json
# spot-check a known registry's creator, name, and description
```

After v1.3.0 (registry count is cumulative):

```bash
nvnmchaind query anchoring registries --output json | jq '.registries | length'
# expect: 2114
```

Cross-check per-registry record counts against `manifest.json`'s `files[].records` for any
registry you want to independently confirm. Reconciliation doesn't depend on transaction logs
or events — these records were written by state migration, not `addRecord` transactions, so
log-scanning indexers will not see them; use keyed/offset-paged `records()` reads instead.
