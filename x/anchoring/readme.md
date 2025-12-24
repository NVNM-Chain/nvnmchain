
# Anchoring Module

## Overview

The Anchoring Module is a crucial component of the Inveniam ecosystem, responsible for managing and storing metadata of documents within the blockchain network. It provides functionality to create, update, and query document metadats, ensuring secure and verifiable document management.

## Key Components

### Keeper

The Keeper is the main component that handles the business logic of the anchoring module. It provides methods for:

- Adding new document metadata
- Removing existing document metadata
- Retrieving document metadata based on various criteria

### Types

The module defines several important types:

1. `Params`: Stores the document metadata module parameters
2. `GenesisState`: Defines the initial state of the module
3. `MsgUpdateParams`: Message for updating document metadata parameters
4. `MsgAddDocument`: Message for adding a new document metadata
5. `Document`: Represents a document metadata associated with a token denom

### Events

The module emits events when document metadata are added or removed. The event types are defined in the types package.

## Usage

To use the anchoring module in your application:

1. Include the module in your app's module configuration.
2. Set up the initial parameters in the genesis state.
3. Use the Keeper methods to interact with the record functionality in your application logic.

## Key Functions

### UpdateParams

The `UpdateParams` function allows for updating the module parameters through a governance proposal or by the designated admin.

### AddDocument

The `AddDocument` function allows users to add new document metadata to the blockchain, associating them with specific token denoms.

For more detailed information on the module's implementation and usage, please refer to the source code and comments within the `x/anchoring` directory.