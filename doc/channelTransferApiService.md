## API of channel transfer

- [rest API](../proto/service.swagger.json)
- [grpc](#grpc)

### GRPC 
(see [proto](../proto/service.proto))

#### TransferByCustomer
- /proto.API/TransferByCustomer

```
Transfer by customer.
Method takes sturcture TransferBeginCustomerRequest as input and returns structure TransferStatusResponse
```

#### TransferByAdmin
- /proto.API/TransferByAdmin

```
Transfer by administartor. 
Method takes structure TransferBeginAdminRequest as input and returns structure TransferStatusResponse 
```

#### MultiTransferByCustomer
- /proto.API/MultiTransferByCustomer

```
Multi transfer by customer.
Method takes sturcture MultiTransferBeginCustomerRequest as input and returns structure TransferStatusResponse
```

#### MultiTransferByAdmin
- /proto.API/MultiTransferByAdmin

```
Multi transfer by administartor. 
Method takes structure MultiTransferBeginAdminRequest as input and returns structure TransferStatusResponse 
```

#### TransferStatus
- /proto.API/TransferStatus

```
Transfer status. 
Method takes structure TransferStatusRequest as input and returns structure TransferStatusResponse
```

#### TransferStatus with filter option
- /proto.API/TransferStatus

```
Receiving the final transfer status, excluding the intermediate state specified in the parameters.
Method takes TransferStatusRequest structure as input, where 
  id_transfer - initialized by transfer ID
  options - initialized by google.protobuf.Option value, where:
   name - contains excludeStatus string
   value - contains string with name of the excluding status (STATUS_IN_PROCESS - for example)  
Method returns TransferStatusResponse structure
```

Example:
```go
    conn, err := grpc.NewClient(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
    sCtx.Require().NoError(err)
    defer conn.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second)
    defer cancel()
    ctx = metadata.AppendToOutgoingContext(ctx, "authorization", token)
    
    transfer := proto.TransferStatusRequest{
        IdTransfer: transferID,
    }
    
    excludeStatus := proto.TransferStatusResponse_STATUS_IN_PROCESS.String()
    value, err := anypb.New(wrapperspb.String(excludeStatus))
    sCtx.Require().NoError(err)
    transfer.Options = append(transfer.Options, &typepb.Option{
        Name:  "excludeStatus",
        Value: value,
    })
    
    out := &proto.TransferStatusResponse{}
    err = conn.Invoke(ctx, "/proto.API/TransferStatus", &transfer, out)
    sCtx.Require().NoError(err)
```