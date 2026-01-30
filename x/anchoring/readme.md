
# Anchoring Module

## Overview

The Anchoring Module manages *registries* and *records* for anchoring off-chain artifacts (e.g., documents) on-chain. A record typically contains a checksum (hash), a URI, optional metadata, and a status. Records are versioned per checksum within a registry.

## Key Components

### Keeper

The Keeper is the main component that handles the business logic and state access for the anchoring module, including:

- Creating registries
- Adding versioned records to registries
- Updating record status
- Looking up registries and records for queries
- Permission checks via RBAC

### RBAC

The module integrates role-based access control (RBAC) to restrict write operations. Roles are scoped either:

- to a registry (by `registry_id`), or
- to a specific record stream (by `checksum`)

Common roles include `admin` and `editor`.

### Types

The module defines several important types:

1. `Params`: Module parameters (includes an `admin` address)
2. `GenesisState`: Initial state of the module
3. `Registry`: A registry of records (id, name, description, creator, created_at)
4. `Record`: An anchored record (registry, uri, checksum, checksum_algo, metadata, timestamp, status, record_id, index, is_latest)

### Messages and Queries

**Transactions (Msg service)**

- `UpdateParams`: Update module params (admin-controlled)
- `AddRegistry`: Create a new registry
- `AddRecord`: Add a new record version under a registry and checksum
- `UpdateRecordStatus`: Update the status of a specific record version
- `GrantRole`: Grant a role to an address (registry-scoped or checksum-scoped)
- `RevokeRole`: Revoke a role from an address

**Queries (Query service)**

- `Params`: Fetch module params
- `Records`: List records filtered by checksum/registry_id/record_id/index (paginated)
- `Registries`: List registries filtered by registry_id/name (paginated)
- `Registry`: Fetch a single registry by id or name

## Usage

To use the anchoring module in your application:

1. Include the module in your app's module configuration.
2. Set up the initial parameters in the genesis state.
3. Use the Msg server methods for state changes and Query server methods for reads.

If you are integrating from an EVM context, use the Anchoring EVM precompile documented below.

## Key Functions

### UpdateParams

The `UpdateParams` function updates module parameters. In practice, this is admin-controlled via `Params.admin`.

### AddRegistry

The `AddRegistry` function creates a new registry.

### AddRecord

The `AddRecord` function adds a new record (and automatically versions it via `record_id` and `index`) under the registry indicated by `record.registry`.

### UpdateRecordStatus

The `UpdateRecordStatus` function updates the `status` field of an existing record version.

For more detailed information on the module's implementation and usage, please refer to the source code and comments within the `x/anchoring` directory.

## EVM Precompile

The Anchoring module exposes an EVM precompile at:

- Address: `0x0000000000000000000000000000000000000a00`

### Complete Human-Readable ABI

The following Solidity-style ABI is generated from the `HumanABI` definitions in `x/anchoring/precompile/anchoring.go`.

```solidity
interface IAnchoringPrecompile {
	struct Record {
		string registry;
		string uri;
		string checksum;
		string checksumAlgo;
		string metadata;
		string timestamp;
		string status;
		uint64 recordId;
		uint64 index;
		bool isLatest;
	}

	struct Registry {
		uint64 id;
		string name;
		string description;
		string creator;
		string createdAt;
	}

	struct PageRequest {
		bytes key;
		uint64 offset;
		uint64 limit;
		bool countTotal;
		bool reverse;
	}

	struct PageResponse {
		bytes nextKey;
		uint64 total;
	}

	function addRegistry(string name, string description)
		external
		returns (uint64 registryId);

	function addRecord(Record record)
		external;

	function updateRecordStatus(uint64 registryId, uint64 recordId, uint64 index, string status)
		external;

	function records(string registry, string checksum, uint64 recordId, uint64 index, PageRequest pagination)
		external
		returns (Record[] records, PageResponse pagination);

	function registries(uint64 registryId, string name, PageRequest pagination)
		external
		returns (Registry[] registries, PageResponse pagination);

	function grantRole(uint64 registryId, string checksum, address account, string role)
		external;

	function revokeRole(uint64 registryId, string checksum, address account, string role)
		external;
}
```

### Function Selectors

Selectors are the first 4 bytes of `keccak256(<function signature>)` and are generated into `x/anchoring/precompile/anchoring.abi.go`.

- `addRecord((string,string,string,string,string,string,string,uint64,uint64,bool))`: `0x9b7b7869`
- `addRegistry(string,string)`: `0x90dee1b1`
- `grantRole(uint64,string,address,string)`: `0xb8fdd1a7`
- `records(string,string,uint64,uint64,(bytes,uint64,uint64,bool,bool))`: `0x02abafdf`
- `registries(uint64,string,(bytes,uint64,uint64,bool,bool))`: `0x15ae270f`
- `revokeRole(uint64,string,address,string)`: `0xacd58bc7`
- `updateRecordStatus(uint64,uint64,uint64,string)`: `0x97b40c25`

### Write Operations (state-changing)

#### addRegistry

- Function signature (for `keccak256`): `addRegistry(string,string)`
- Function selector: `0x90dee1b1`
- Inputs: `(string name, string description)`
- Parameter encoding (Ethereum ABI):
	- Call data = selector (4 bytes) + `abi.encode(name, description)`
	- Each `string` is dynamic: head contains a 32-byte offset; tail contains `length (uint256)` + UTF-8 bytes + zero padding to a 32-byte boundary.
- Return value encoding: returns `(uint64 registryId)` encoded as a single 32-byte word (left-padded).
- Expected gas (rough): ~`80,000–250,000` EVM gas (depends on KV writes and string lengths)
- Authorization checks:
	- No RBAC permission check; effectively any EVM caller can create a registry as long as `name` is unique.
- State mutations:
	- Creates `registryId = RegistryCount + 1`
	- Stores `Registries[registryId] = {id, name, description, creator, created_at}`
	- Stores `RegistryIdByName[name] = registryId`
	- Initializes RBAC: sets the registry admin role to be its own admin; adds the creator as a member of the registry admin role
	- Initializes registry record counter; updates `RegistryCount`
- Events emitted:
	- No anchoring-module-specific events are emitted in code; only generic Cosmos SDK tx events.

#### addRecord

- Function signature (for `keccak256`): `addRecord((string,string,string,string,string,string,string,uint64,uint64,bool))`
- Function selector: `0x9b7b7869`
- Inputs: `Record record` where:
	- `Record = (string registry, string uri, string checksum, string checksumAlgo, string metadata, string timestamp, string status, uint64 recordId, uint64 index, bool isLatest)`
- Parameter encoding (Ethereum ABI):
	- Call data = selector + `abi.encode(record)`
	- Tuples are encoded like structs: offsets for dynamic fields (`string`s) plus 32-byte words for static fields (`uint64`, `bool`).
	- `uint64` encodes as 32-byte left-padded; `bool` as `0`/`1` in a 32-byte word.
	- Note: although `Record` includes `timestamp`, `recordId`, `index`, `isLatest`, the chain overwrites some of these on write.
- Return value encoding: returns `()` (empty return data)
- Expected gas (rough): ~`120,000–450,000` EVM gas (depends on string sizes and whether this is a new checksum vs a new version)
- Authorization checks:
	- Caller must have `admin` or `editor` via one of:
		- checksum-scoped role, or
		- registry-scoped role, or
		- global role.
- State mutations:
	- Resolves `registryId` from `record.registry` name
	- Determines/assigns `recordId` for `(registryId, checksum)`; increments per-registry record counters when needed
	- Increments per-record `index` and sets:
		- `record.Timestamp = blockTime`
		- `record.Index = index`
		- `record.IsLatest = true`
	- Stores the record at key `(registryId, recordId, index)`
	- Updates indexes/mappings for `checksum ↔ recordId`
	- If `index > 1`, marks the previous version `(index-1)` as `is_latest = false`
- Events emitted:
	- No anchoring-module-specific events are emitted in code; only generic Cosmos SDK tx events.

#### updateRecordStatus

- Function signature (for `keccak256`): `updateRecordStatus(uint64,uint64,uint64,string)`
- Function selector: `0x97b40c25`
- Inputs: `(uint64 registryId, uint64 recordId, uint64 index, string status)`
- Parameter encoding (Ethereum ABI):
	- Call data = selector + `abi.encode(registryId, recordId, index, status)`
	- `uint64` are static 32-byte words; `string status` is dynamic (offset + length + bytes).
- Return value encoding: returns `()` (empty return data)
- Expected gas (rough): ~`70,000–220,000` EVM gas (read + write)
- Authorization checks:
	- Loads the record to obtain its `checksum`, then requires caller to have `admin` or `editor` (checksum-scoped OR registry-scoped OR global).
- State mutations:
	- Loads record at `(registryId, recordId, index)`
	- Sets `record.Status = status`
	- Writes it back to `(registryId, recordId, index)`
- Events emitted:
	- No anchoring-module-specific events are emitted in code; only generic Cosmos SDK tx events.

#### grantRole

- Function signature (for `keccak256`): `grantRole(uint64,string,address,string)`
- Function selector: `0xb8fdd1a7`
- Inputs: `(uint64 registryId, string checksum, address account, string role)`
- Parameter encoding (Ethereum ABI):
	- Call data = selector + `abi.encode(registryId, checksum, account, role)`
	- `address` is 20 bytes left-padded to 32 bytes; `string` params are dynamic.
- Return value encoding: returns `()` (empty return data)
- Expected gas (rough): ~`80,000–250,000` EVM gas (RBAC admin lookup + membership write)
- Authorization checks:
	- Only an address with the registry admin role for `registryId` can grant roles (enforced by RBAC).
	- Scope selection:
		- If `checksum != ""`: checksum-scoped role id = `keccak256("<checksum>:<role>")`
		- Else: registry-scoped role id = `keccak256("registry:<registryId>:<role>")`
	- Admin role for any granted role is set to the registry admin role.
- State mutations:
	- Sets/updates `RoleAdmins[role] = registryAdminRole`
	- Adds `RoleMembers[(role, account)] = {}`
- Events emitted:
	- No anchoring-module-specific events are emitted in code; only generic Cosmos SDK tx events.

#### revokeRole

- Function signature (for `keccak256`): `revokeRole(uint64,string,address,string)`
- Function selector: `0xacd58bc7`
- Inputs: `(uint64 registryId, string checksum, address account, string role)`
- Parameter encoding (Ethereum ABI):
	- Call data = selector + `abi.encode(registryId, checksum, account, role)`
- Return value encoding: returns `()` (empty return data)
- Expected gas (rough): ~`70,000–230,000` EVM gas (depends on membership existence and role(s) attempted)
- Authorization checks:
	- Only an address with the registry admin role for `registryId` can revoke roles (enforced by RBAC).
	- If `role == ""`, the implementation attempts to revoke `editor` first, then `admin` (within the selected scope).
- State mutations:
	- Removes `RoleMembers[(role, account)]` for the computed role (registry- or checksum-scoped)
	- Does not delete the role admin mapping.
- Events emitted:
	- No anchoring-module-specific events are emitted in code; only generic Cosmos SDK tx events.

### Read Operations (view)

#### records

- Function signature (for `keccak256`): `records(string,string,uint64,uint64,(bytes,uint64,uint64,bool,bool))`
- Function selector: `0x02abafdf`
- Return value encoding: returns `(Record[] records, PageResponse pagination)` encoded per standard Ethereum ABI rules for dynamic arrays/tuples.
- State queried:
	- If `registry != ""`, resolves registry name → `registryId` via `RegistryIdByName`.
	- Queries Cosmos module state via the anchoring gRPC query server (`Query/Records`) with filters `checksum`, `registry_id`, `record_id`, `index`, and pagination.
- Data source: Cosmos state (Cosmos SDK KV/collections), not EVM contract storage.
- Staleness: within a single EVM tx, it should observe up-to-date Cosmos cached state because the precompile commits the cache context before executing each call.

Example output (mapped to `Record` field names):

```json
[
	{
		"registry": "test-registry",
		"uri": "ipfs://abc123",
		"checksum": "abc123",
		"checksumAlgo": "sha256",
		"metadata": {"document": "Record 1 v2", "figi": "", "individualId": ""},
		"timestamp": "2026-01-30 05:39:59.971631 +0000 UTC",
		"status": "",
		"recordId": 1,
		"index": 2,
		"isLatest": true
	},
	{
		"registry": "test-registry",
		"uri": "ipfs://def456",
		"checksum": "def456",
		"checksumAlgo": "sha256",
		"metadata": {"document": "Record 2", "figi": "", "individualId": ""},
		"timestamp": "2026-01-30 05:40:01.228424 +0000 UTC",
		"status": "",
		"recordId": 2,
		"index": 1,
		"isLatest": true
	}
]
```

Note: on-chain (ABI) the `metadata` field is returned as a JSON-encoded `string`; the example shows it parsed as JSON for readability.

#### registries

- Function signature (for `keccak256`): `registries(uint64,string,(bytes,uint64,uint64,bool,bool))`
- Function selector: `0x15ae270f`
- Return value encoding: returns `(Registry[] registries, PageResponse pagination)` encoded per standard Ethereum ABI rules for dynamic arrays/tuples.
- State queried:
	- Queries Cosmos module state via the anchoring gRPC query server (`Query/Registries`) with filters `registry_id`, `name`, and pagination.
- Data source: Cosmos state (Cosmos SDK KV/collections), not EVM storage.
- Staleness: same semantics as `records` for the current EVM tx.

Example output (mapped to `Registry` field names):

```json
[
	{
		"id": 4,
		"name": "query-specific-reg",
		"description": "query-specific-reg",
		"creator": "inveniam1x7x9pkfxf33l87ftspk5aetwnkr0lvlvyde8p7",
		"createdAt": "2026-01-30 05:40:15.090584 +0000 UTC"
	}
]
```
