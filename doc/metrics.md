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
Type: counter; Information about channel-transfer application\
Indicators:
- channel - application version 
- chaincode - sdk fabric version

 **2.** **total_transfer_created**\
Type: counter; Quantity of created transfers\
Indicators:
- channel - channel-transfer's destination channel

Description:
- quantity of initiated transfers by channel since app was launched

 **3.** **total_reconnects_to_fabric**\
Type: counter; Quantity of reconnections to HLF\
Indicators:
- channel - channel-transfer's destination channel

Description:
- Quantity of reconnection to HLF by channel

 **4.** **total_success_transfer**\
Type: counter; Quantity of successfully completed transfers\
Indicators:
- channel - channel-transfer's destination channel

Description:
- Quantity of successfully completed transfers by channel

**5.** **total_failure_transfer**\
Type: counter; Quantity of transfers that completed with error\
Indicators:
- channel - channel-transfer's destination channel
- fail Transfer Tag - error tag: timeout (expired transfer), source channel (transfer from), destination channel (transfer to)

Description:
- Quantity of failed transfers with tags - reasons of unsuccessful completion since app was launched. 

 **6.** **total_in_work_transfer**\
Type: probe; Quantity of processing transfers\
Indicators:
- channel - channel-transfer's destination channel
- transferStatus - transfer status

Description:
- Quantity of processing transfers by channels and transfer statuses (five stages of processing)

 **7.** **application_init_duration**\
Type: probe; service's status: application intialization duration\

Description:
- application intialization duration, time to start an application in seconds

**8.** **fabric_connection_status**\
Type: probe; HLF connection status\
Indicators:

- channel - channel-transfer's destination channel

Description:
- Statuses - if channel is connected to HLF or not

**9.** **collector_process_block_num**\
Type: probe; Block number, processed by collector\
Indicators:
- channel - channel-transfer's destination channel

Description:
- Current number of the block, being processed by collector during transfers formation by channels

**10.** **time_duration_complete_transfer_before_responding**\
Type: bar chart; time in seconds from the start of the transfer until the notification is sent to user\
Indicators:
- channel - channel-transfer's destination channel

Description:
- time spent completing the first two stages of the transfer operation (the first two transactions), after which  the user is notified of the transfer execution

**11.** **transfer_execution_time_duration**\
Type: bar chart; time from the start of transfer operation execution until the operation completion\
Indicators:
- channel - channel-transfer's destination channel

Description:
- time spent to complete the transfer from the moment of record is created in state until it's deletion

**12.** **time_duration_transfer_stage_execution**\
Type: bar chart; transfer stages execution duration\
Indicators:
- channel - channel-transfer's destination channel
- transferStatus - transfer status

Description:
- time spent on each stage of the transfer by channels