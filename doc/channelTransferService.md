## Описание сервиса 

### Инициализация сервиса

```mermaid
sequenceDiagram
    ChannelTransferService ->> ChannelTransferService: проверка, что в параметрах конфигурации заданы каналы
    
    ChannelTransferService ->> hlf: проверка подключения
    ChannelTransferService ->> redis: проверка подключения 
    
    ChannelTransferService ->> redis: запросить номер последнего обработанного блока по каждому каналу
    
```

### Подписка на блоки ledger

Получаем данные из ledger по всем каналам, заданным в конфигурации сервиса и найденным в параметре TO трансферов, и актуализируем данные в redis. 
Сканирование каналов ledger начинается или с нулевого блока, или с последнего проанализированного сервисом блока,
информация о котором автоматически сохраняется в redis. 

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: получать следующий по порядку блок
    ChannelTransferService ->> hlf: распарсить блок
    ChannelTransferService ->> hlf: проверка статусов транзакций блока на валидность
    ChannelTransferService ->> hlf: перебрать валидные транзакции, отфильтровать активные транзакции трансфера
    ChannelTransferService ->> redis: обновить (или добавить) данные о активных трансферах по найденным в ledger транзакциям
    ChannelTransferService ->> hlf: ожидание нового блока

```

###  Синхронизация статусов трансферов в redis с ledger для каждого канала

Процесс проверки выполнения транзакций переводов и актуализации статусов в redis

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: получить список не активных трансферов канала
    ChannelTransferService ->> ChannelTransferService: для каждого трансфера по транзакциям ledger определить статус
    ChannelTransferService ->> ChannelTransferService: обновить значение статуса в redis

```

### Определение статуса трансфера

Процесс вычисляет текущий статус трансфера на основе данных ledger 

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: выполняем запрос сhannelTransfersFrom в ledger
    ChannelTransferService ->> ChannelTransferService: определяем валидность канала TO
    ChannelTransferService ->> hlf: выполняем запрос сhannelTransferTo в ledger
    ChannelTransferService ->> hlf: если запрос к метаданным трансфера в канале TO вернул данные, то проверяем наличие транзакции batchExecute 
    ChannelTransferService ->> ChannelTransferService: на основе полученных данных вычисляем текущий статус трансфера  
```

### Обработка запроса к API на создание трансфера. 1 часть

```mermaid
sequenceDiagram
    external ->> ChannelTransferService: проверка полей запроса
    ChannelTransferService ->> redis: сохранить запрос
    ChannelTransferService ->> hlf: выполнить channelTransferByCustomer/channelTransferByAdmin
```

### Завершение 1-й части трансфера - проверка batchExecute в канале FROM

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: проверить batchExecute и результат выполнения транзакции по созданию трансфера
    ChannelTransferService ->> hlf: если время выполнения истекло (timeout) или возникла ошибка - выполняется invoke cancelCCTransferFrom
    ChannelTransferService ->> redis: если время выполнения истекло (timeout) или возникла ошибка - изменяем статус трансфера и заканчиваем его обработку 

```
### Создание 2-й части трансфера - перевод в канал TO

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: выполнить invoke createCCTransferTo

```

### Завершение 2-й части трансфера - проверка batchExecute в канале TO

```mermaid
sequenceDiagram
    ChannelTransferService ->> redis: проверить batchExecute и результат выполнения транзакции по созданию трансфера
    ChannelTransferService ->> hlf: если время выполнения истекло (timeout) или возникла ошибка - выполняется invoke cancelCCTransferFrom
    ChannelTransferService ->> redis: если время выполнения истекло (timeout) или возникла ошибка - изменяем статус трансфера и заканчиваем его обработку 
```
### 3-я часть трансфера - выставляем признак 2-й фазы

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: выполняется invoke commitCCTransferFrom
    
```

### 4-я часть трансфера - удалить метаданные трансфера в канале TO

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: выполняется invoke cancelTransferTo
    
```

### 5-я часть трансфера - удалить метаданные трансфера в канале FROM

```mermaid
sequenceDiagram
    ChannelTransferService ->> hlf: выполняется invoke cancelTransferFROM

```
