## Metrics
Metrics are available at `/metrics`\
The channel-transfer service provides these metrics:


- **app_info**\
  _counter_,  application info\
  Labels:\
  _channel_ - channel-transfer's version\
  _chaincode_ - fabric SDK version
  
- **total_transfer_created**\
  _counter_, Number of created transfers\
  Labels:\
  _channel_ - channel-transfer's processed channel

- **total_reconnects_to_fabric**\
  _counter_, number of reconnect to HLF\
  Labels:\
  _channel_ - channel-transfer's processed channel

- **total_success_transfer**\
  _counter_, Number success of created transfers\
  Labels:\
  _channel_ - channel-transfer's processed channel
  
- **total_failure_transfer**\
  _counter_, Number of failure ended transfers\
  Labels:\
  _channel_ - channel-transfer's processed channel\
  _failTransferTag_ - expired transfer, transfer from, transfer to
  
- **total_in_work_transfer**\
  _gauge_, Number of transfers in work\
  Labels:\
  _channel_ - channel-transfer's processed channel\
  _transferStatus_ - transfer status

- **application_init_duration**\
  _gauge_, General service status: application init duration
  
- **fabric_connection_status**\
  _gauge_, HLF connection status\
  Labels:\
  _channel_ - channel-transfer's processed channel
  
- **collector_process_block_num**\
  _gauge_, Block number processed by the collector\
  Labels:\
  _channel_ - channel-transfer's processed channel

- **time_duration_complete_transfer_before_responding**\
  _histogram_, Time duration to complete the transfer before responding to the user in seconds\
  Labels:\
  _channel_ - channel-transfer's processed channel
  
- **transfer_execution_time_duration**\
  _histogram_, Transfer execution time duration from start to finish in seconds\
  Labels:\
  _channel_ - channel-transfer's processed channel

- **time_duration_transfer_stage_execution**\
  _histogram_, Time duration of transfer stage execution \
  Labels:\
  _channel_ - channel-transfer's processed channel\
  _transferStatus_ - transfer status


## Detailed description

**1.**   **app_info**\
Тип: счетчик; информация о приложении channel-transfer\
Показатели:
- channel - версия приложения 
- chaincode - версия sdk fabric

 **2.** **total_transfer_created**\
Тип: счетчик; количество сформированных трансферов\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- количество инициированных трансферов с момента запуска приложения в разрезе каналов

 **3.** **total_reconnects_to_fabric**\
Тип: счетчик; количество переподключений к HLF\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- число повторных подключений  приложения к HLF в разрезе каналов

 **4.** **total_success_transfer**\
Тип: счетчик; число успешно завершенных трансферов\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- количество успешно завершенных трансферов в разрезе каналов

**5.** **total_failure_transfer**\
Тип: счетчик; число трансферов, завершившихся с ошибкой\
Показатели:
- channel - канал назначения channel-transfer
- fail Transfer Tag - признак ошибки: истек таймаут (expired transfer), канал-источник (transfer from), канал-получатель (transfer to)

Описание:
- Количество (с момента запуска сервиса) неуспешных трансферов с тегами - причинами незавершения трансфера. 

 **6.** **total_in_work_transfer**\
Тип: датчик; число трансферов, находящихся в обработке\
Показатели:
- channel - канал назначения channel-transfer
- transferStatus - статус трансфера

Описание:
- число трансферов, находящихся в обработке,  в разрезе по каналам и статусам трансфера (пяти этапам обработки)

 **7.** **application_init_duration**\
Тип: датчик; общий статус сервиса: продолжительность инициализации приложения\

Описание:
- продолжительность инициализации приложения, время в секундах, потребовавшееся для старта приложения

**8.** **fabric_connection_status**\
Тип: датчик; статус подключения к HLF\
Показатели:

- channel - канал назначения channel-transfer

Описание:
- статусы - канал подключен/не подключен к HLF

**9.** **collector_process_block_num**\
Тип: датчик; номер блока, обрабатываемый коллектором\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- текущий номер блока, обрабатываемый коллектором в ходе формирования трансферов (в разрезе каналов)

**10.** **time_duration_complete_transfer_before_responding**\
Тип: гистограмма; время в секундах от начала выполнения трансфера до отправки извещения пользователю\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- время, затраченное на выполнение первых двух этапов трансфера (двух первых транзакций), после которых пользователю направляется извещение о исполнении трансфера 

**11.** **transfer_execution_time_duration**\
Тип: гистограмма; время от начала выполнения трансфера до завершения выполнения\
Показатели:
- channel - канал назначения channel-transfer

Описание:
- время, затраченное на полное выполнение трансфера (от момента появления записи в state до момента ее удаления)

**12.** **time_duration_transfer_stage_execution**\
Тип: гистограмма; время выполнения этапов трансфера\
Показатели:
- channel - канал назначения channel-transfer
- transferStatus - статус трансфера

Описание:
- время, затраченное на выполнение каждого из этапов трансфера в разрезе каналов