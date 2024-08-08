### Smart contract methods for transfer execution

**LINK:** [token methods](https://github.com/anoideaopen/foundation/-/blob/master/core/chtransfer.go#L26)

#### INVOKE

- channelTransferByCustomer (batch) - transfer creation for client
- channelTransferByAdmin (batch) - transfer creation for admin

- channelMultiTransferByCustomer (batch) - transfer creation for client
- channelMultiTransferByAdmin (batch) - transfer creation for admin
 
- createCCTransferTo - create transfer to destination channel

- deleteCCTransferFrom - delete transfer from source channel
- deleteCCTransferTo - delete transfer from destination channel

- commitCCTransferFrom (no batch) - commit transfer in source channel

- cancelCCTransferFrom - delete transfer in source channel and return tokens 
- cancelTransferTo - delete transfer in destination channel and return tokens

#### QUERY

- channelTransferFrom - returns transfer data from source channel
- channelTransferTo - returns transfer data from destination channel
- channelTransfersFrom - returns paginated data of all processing transfers in source channel  


### Changes to World State ledger made by transactions
**LINK:** [CCTransfer](https://github.com/anoideaopen/foundation/-/blob/master/proto/batch.proto#L231)

| method                         | batch <br/> transaction  | transfer metadata                                     | balance to be changed                         |
|--------------------------------|:------------------------:|-------------------------------------------------------|-----------------------------------------------|
| channelTransferByCustomer      |           yes            | saving CCTransfer structure in channel FROM           | TokenBalance & GivenBalance or AllowedBalance |
| channelTransferByAdmin         |           yes            | saving CCTransfer structure in channel FROM           | TokenBalance & GivenBalance or AllowedBalance |
| channelMultiTransferByCustomer |           yes            | saving CCTransfer structure in channel FROM           | TokenBalance & GivenBalance or AllowedBalance |
| channelMultiTransferByAdmin    |           yes            | saving CCTransfer structure in channel FROM           | TokenBalance & GivenBalance or AllowedBalance |
| createCCTransferTo             |           yes            | saving CCTransfer structure in channel TO             | AllowedBalance & TokenBalance or GivenBalance |
| commitCCTransferFrom           |                          | setting IsCommit in CCTransfer in channel FROM        |                                               |
| deleteCCTransferTo             |                          | deleting CCTransfer structure in channel TO           |                                               |
| deleteCCTransferFrom           |                          | deleting CCTransfer structure in channel FROM         |                                               |
| cancelCCTransferFrom           |           yes            | deleting CCTransfer structure in channel FROM         | TokenBalance & GivenBalance or AllowedBalance |
