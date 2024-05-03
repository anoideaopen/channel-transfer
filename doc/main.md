# System components description

```mermaid
flowchart TD
    user --> channelTransferApiService
    channelTransferApiService --> redis   
    channelTransferApiService --> hlf   
    channelTransferService --> redis 
    hlf --> channelTransferService 
    channelTransferService --> hlf 
```

- [channel transfer chaincode API](./chaincodeAPI.md)
- [domain model](./domainModel.md)
- [channel transfer API service](./channelTransferApiService.md)
- [channel transfer service](./channelTransferService.md)
