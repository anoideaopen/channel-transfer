# Описание компонентов системы

```mermaid
flowchart TD
    user --> channelTransferApiService
    channelTransferApiService --> redis   
    channelTransferApiService --> hlf   
    channelTransferService --> redis 
    hlf --> channelTransferService 
    channelTransferService --> hlf 
```

- [channel transfer chaincode API](channelTransferChaincodeAPI.md)
- [domain model](./domainModel.md)
- [channel transfer API service](./channelTransferApiService.md)
- [channel transfer service](./channelTransferService.md)
