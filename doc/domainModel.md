# Domain model

Context: Identity
----

### User

- name
- public_key
- private_key
- signature

### Role

- CUSTOMER
- ADMIN

Context: API Transfer token between channels
----

### Admin

### Customer

### Channel transfer FROM ADMIN.

```json
{
  "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
  "customer_id": "customer_id",
  "tokenTo": "destination channel",
  "contractFrom": "user From",
  "contractTo": "user To",
  "amount": "amount",
  "nonce": "nonce",
  "publicKey": "public key",
  "signMessage": "request sign message",
  "methodName": " - method name",
  "requestID": " - request id (may be empty)",
  "chaincode": " - chaincode name",
  "channel": " - channel name",
  "args": "... - user's arguments"
}
```

### Channel transfer FROM CUSTOMER.

```json
{
  "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
  "customer_id": "customer_id",
  "tokenTo": "destination channel",
  "contractFrom": "user From",
  "contractTo": "user To",
  "amount": "amount",
  "nonce": "nonce",
  "publicKey": "public key",
  "signMessage": "request sign message",
  "methodName": " - method name",
  "requestID": " - request id (may be empty)",
  "chaincode": " - chaincode name",
  "channel": " - channel name",
  "args": "... - user's arguments"
}
```

### Channel transfer TO.

```json
{
  "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953"
}
```

### Channel Transfer Process

### Channel Transfer Process Status

Statuses user should see in response from ChannelTransferAPI

- IN_PROGRESS - operation processing without errors
- ERROR - error occurs on any transfer operation stage (funds may get stuck, error proceedings may take time)
- COMPLETED - transfer operation successfully completed, all records are deleted from state (TransferFrom, TransferTo)
- CANCELED - transfer operation is cancelled by timeout or error, all records are deleted from state (TransferFrom, TransferTo)

Context: Process Transfer token between channels
----

### Ledger info

Parsing blocks and transactions info

```json
{
  "channel": "ba",
  "block_id": "1",
  "tx_id": "1"
}
```

### Channel transfer state

```json
{
  "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
  "last_transfer_info": {
    "tx_id": "tx3",
    "block_id": "4",
    "channelName": "ba02",
    "chaincodeName": "ba02",
    "methodName": "TransferTo",
    "type": "batchExecute",
    "status": "ERROR",
    "error_message": "Error message"
  },
  "transfer_info": {
    "from": {
      "channelName": "ba",
      "chaincodeName": "ba"
    },
    "to": {
      "channelName": "ba02",
      "chaincodeName": "ba02"
    }
  }
}
```

### Channel transfer state history

INDEX: channelName + chaincodeName + request_id + tx_id

There can be several batches for one preimage, so need to analyze only successfully  batchExecute

```json
[
  {
    "tx_id": "tx0",
    "block_id": "1",
    "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
    "channelName": "ba",
    "chaincodeName": "ba",
    "methodName": "TransferFrom",
    "type": "preimage",
    "status": "COMPLETED",
    "retry": 0
  },
  {
    "tx_id": "tx1",
    "block_id": "2",
    "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
    "channelName": "ba",
    "chaincodeName": "ba",
    "methodName": "TransferFrom",
    "type": "batchExecute",
    "status": "COMPLETED",
    "retry": 0
  },
  {
    "tx_id": "tx2",
    "block_id": "3",
    "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
    "channelName": "ba02",
    "chaincodeName": "ba02",
    "methodName": "TransferTo",
    "type": "preimage",
    "status": "COMPLETED",
    "retry": 0
  },
  {
    "tx_id": "tx3",
    "block_id": "4",
    "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
    "channelName": "ba02",
    "chaincodeName": "ba02",
    "methodName": "TransferTo",
    "type": "batchExecute",
    "status": "ERROR",
    "error_message": "Error message",
    "retry": 0
  }
]
```

### process_status

Errors for preimage & batchExecute are combined, detailed error description is placed to an "error_message" field:

- IN_PROGRESS_TransferFrom
- IN_PROGRESS_TransferTo
- IN_PROGRESS_TransferFromDelete
- IN_PROGRESS_TransferToDelete
- COMPLETED_TransferFrom
- COMPLETED_TransferTo
- COMPLETED_TransferFromDelete
- COMPLETED_TransferToDelete
- ERROR_TransferFrom
- ERROR_TransferTo
- ERROR_TransferFromDelete
- ERROR_TransferToDelete
- COMPLETED - transfer successfully completed, all state records are deleted (TransferFrom, TransferTo)
- CANCELED - transfer is cancelled by timeout or error, all state records are deleted (TransferFrom, TransferTo)
