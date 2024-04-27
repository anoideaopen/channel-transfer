## Service description 

### Service initialization

```mermaid
sequenceDiagram
    ChannelTransferService ->> ChannelTransferService: check for channels are specified in the initialization parameters
    
    ChannelTransferService ->> hlf: checking connection
    ChannelTransferService ->> redis: checking connection 
    
    ChannelTransferService ->> redis: requesting number of the last processed block for each channel
    
```

### Subscription to ledger blocks

Receiving data from ledger through all channels specified in the service configuration parameters and found specified in the “TO” transfers parameter, updating data in Redis. 
Ledger channel scanning starts from the zero block or from the last block analyzed by the service, information about which is automatically saved in Redis. 

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: getting the next block in order
    ChannelTransferService ->> hlf: parsing block
    ChannelTransferService ->> hlf: checking block transaction statuses for validity
    ChannelTransferService ->> hlf: enumeration of valid transactions, filtering of active transfer transactions
    ChannelTransferService ->> redis: updating or adding data on active transfers for transactions found in the ledger
    ChannelTransferService ->> hlf: awaiting for the next block

```

###  Syncing transfer statuses in redis with ledger for each channel

Checking the completion of transfer transactions and updating statuses in Redis

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: getting a list of inactive channel transfers
    ChannelTransferService ->> ChannelTransferService: determining the status for each transfer for ledger transactions
    ChannelTransferService ->> ChannelTransferService: updating status value in Redis

```

### Determining transfer status

Calculating current transfer status basing on ledger data

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: requesting сhannelTransfersFrom in ledger
    ChannelTransferService ->> ChannelTransferService: checking validity of "TO"-channel
    ChannelTransferService ->> hlf: requesting сhannelTransferTo in ledger
    ChannelTransferService ->> hlf: checking for the presence of a batchExecute transaction if a request to transfer metadata in the "TO"-channel returns valid data
    ChannelTransferService ->> ChannelTransferService: calculating the current transfer status based on the received data

```

### Processing transfer's 1st stage - requesting to the API for transfer creation

```mermaid
sequenceDiagram
    external ->> ChannelTransferService: checking request fields
    ChannelTransferService ->> redis: saving request
    ChannelTransferService ->> hlf: executing channelTransferByCustomer/channelTransferByAdmin

```

### Completion of transfer's 1st stage - checking batchExecute in the "FROM"-channel

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: checking batchExecute and the result of create transfer transaction execution
    ChannelTransferService ->> hlf: in case of timeout or error executing invoke cancelCCTransferFrom
    ChannelTransferService ->> redis: in case of timeout or error changing transfer status and processing finishes 

```

### Processing transfer's 2nd stage - transferring to "TO"-channel

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: executing invoke createCCTransferTo

```

### Completion of transfer's 2nd stage - checking batchExecute in the "TO"-channel

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: checking batchExecute calculating the current transfer status based on the received data
    ChannelTransferService ->> hlf: in case of timeout or error executing invoke cancelCCTransferFrom
    ChannelTransferService ->> redis: in case of timeout or error changing transfer status and processing finishes

```

### Processing transfer's 3rd stage - setting the 2nd phase sign

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: executing invoke commitCCTransferFrom
    
```

### Processing transfer's 4th stage - deleting transfer metadata in "TO"-channel

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: executing invoke cancelTransferTo
    
```

### Processing transfer's 5th stage - deleting transfer metadata in "FROM"-channel

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: executing invoke cancelTransferFROM

```
