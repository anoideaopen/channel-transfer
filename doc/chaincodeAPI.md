### Методы в смарт контракте для выполнения перевода в другой канал

**LINK:** [token methods](https://github.com/anoideaopen/foundation/-/blob/master/core/chtransfer.go#L26)

#### INVOKE

- channelTransferByCustomer (batch) - создание перевода для клиента
- channelTransferByAdmin (batch) - создание перевода для администратора
 
- createCCTransferTo - перевод в канал куда происходит перевод

- deleteCCTransferFrom - удалить исходный трансфер
- deleteCCTransferTo - удалить трансфер из канала получения перевода

- commitCCTransferFrom (no batch) - отметить трансфер в канале из которого выполняется перевод

- cancelCCTransferFrom - удалить исходный трансфер и делает возврат токенов
- cancelTransferTo - удалить трансфер из канала получения перевода и делает возврат токенов

#### QUERY

- channelTransferFrom - возвращает данные о трансфере из канала From
- channelTransferTo - возвращает данные о трансфере из канала To
- channelTransfersFrom - возвращает данные о всех активных трансферах канала From, данные отдает с пагинацией 


### Изменения производимые транзакциями трансфера в World State ledger
**LINK:** [CCTransfer](https://github.com/anoideaopen/foundation/-/blob/master/proto/batch.proto#L231)

| метод                     | batch <br/> транзакция | метаданные трансфера                                         | изменяемый баланс                              |
|---------------------------|:----------------------:|--------------------------------------------------------------|------------------------------------------------|
| channelTransferByCustomer |           да           | сохраняем структуру CCTransfer в канале FROM                 | TokenBalance и GivenBalance или AllowedBalance |
| channelTransferByAdmin    |           да           | сохраняем структуру CCTransfer канале FROM                   | TokenBalance и GivenBalance или AllowedBalance |
| createCCTransferTo        |           да           | сохраняем структуру  канале TO                               | AllowedBalance или TokenBalance и GivenBalance |
| commitCCTransferFrom      |                        | изменяем атрибут IsCommit структуры CCTransfer в канале FROM |                                                |
| deleteCCTransferTo        |                        | удаляем структуру CCTransfer канала TO                       |                                                |
| deleteCCTransferFrom      |                        | удаляем структуру CCTransfer канала FROM                     |                                                |
| cancelCCTransferFrom      |           да           | удаляем структуру CCTransfer канала FROM                     | TokenBalance и GivenBalance или AllowedBalance |
