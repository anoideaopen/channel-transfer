## API of channel transfer

- [rest API](../proto/service.swagger.json)
- [grpc](#grpc)

### GRPC 
(см. [proto](../proto/service.proto))

#### TransferByCustomer
- /proto.API/TransferByCustomer

```
Трансфер от пользователя.
Метод принимает структуру TransferBeginCustomerRequest и возвращает структуру TransferStatusResponse
```

#### TransferByAdmin
- /proto.API/TransferByAdmin

```
Трансфер от администратора. Метод принимает структуру TransferBeginAdminRequest и возвращает структуру TransferStatusResponse 
```

#### TransferStatus
- /proto.API/TransferStatus

```
Получение статуса трансфера. Метод принимает структуру TransferStatusRequest и возвращает структуру TransferStatusResponse
```

#### TransferStatus with filter option
- /proto.API/TransferStatus

```
Получение итогового статуса трансфера, исключая промежуточное состояние, заданное в параметрах.
Метод принимает структуру TransferStatusRequest, где 
  id_transfer - инициализирован идентификатором трансфера
  options - инициализирован значением google.protobuf.Option, где:
   name - содержит строку excludeStatus
   value - содержит строку с названием исключаемого статуса: STATUS_IN_PROCESS  
Метод возвращает структуру TransferStatusResponse
```

Пример:
```go
    conn, err := grpc.Dial(targetGrpc, grpc.WithTransportCredentials(transportCredentials))
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