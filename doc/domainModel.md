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
  "tokenTo": "в какой токен перевести",
  "contractFrom": "от кого перевести",
  "contractTo": "кому перевести",
  "amount": "сколько",
  "nonce": "нонс",
  "publicKey": "публичный ключ",
  "signMessage": "подпись запроса",
  "methodName": " - название метода",
  "requestID": " - id запроса (может быть пустым)",
  "chaincode": " - имя чейнкода",
  "channel": " - имя канала",
  "args": "... - все пользовательские аргументы",
  "перевод": " владельцем"
}
```

### Channel transfer FROM CUSTOMER.

```json
{
  "request_id": "6c0bfd08-bf56-440f-8ead-46c996c10953",
  "customer_id": "customer_id",
  "tokenTo": "в какой токен перевести",
  "contractFrom": "от кого перевести",
  "contractTo": "кому перевести",
  "amount": "сколько",
  "nonce": "нонс",
  "publicKey": "публичный ключ",
  "signMessage": "подпись запроса",
  "methodName": " - название метода",
  "requestID": " - id запроса (может быть пустым)",
  "chaincode": " - имя чейнкода",
  "channel": " - имя канала",
  "args": "... - все пользовательские аргументы",
  "перевод": " владельцем"
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

Только эти статусы должен видеть клиент в ответах от сервиса ChannelTransferAPI

- IN_PROGRESS - процесс выполнения перевода идет без ошибок
- ERROR - возникла ошибка на любой стадии перевода (денежные средства могут зависнуть, разбирательство по ошибки требует
  время)
- COMPLETED - весь трансфер прошел и удален стейт (TransferFrom, TransferTo)
- CANCELED - выполнилась отмена по таймауту или ошибки и удален стейт (TransferFrom, TransferTo)

Context: Process Transfer token between channels
----

### Ledger info

Информация о парсинге блоков и транзакций

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
    "error_message": "Описание ошибки"
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

На 1 preimage может быть несколько батчей, по этой причине смотрим только успешно выполненную транзакцию batchExecute

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
    "error_message": "Описание ошибки",
    "retry": 0
  }
]
```

### process_status

Объединяем ошибки для preimage и batchExecute, подробное описание ошибки выносим в отдельное поле "error_message"

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
- COMPLETED - весь трансфер прошел и удален стейт (TransferFrom, TransferTo)
- CANCELED - выполнилась отмена по таймауту или ошибки и удален стейт (TransferFrom, TransferTo)
