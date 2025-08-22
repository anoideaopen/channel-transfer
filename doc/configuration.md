# Configuration

## TOC
- [Environments](#environments)
- [Configuration file](../pkg/config/config_test.yaml)

## Environments

#### CHANNEL_TRANSFER_LOGLEVEL
- values: debug, error, info, warn
#### CHANNEL_TRANSFER_LOGTYPE
- values: console, json, gcp
#### CHANNEL_TRANSFER_PROFILEPATH
- path to Fabric connection profile 
#### CHANNEL_TRANSFER_HLF_CONNECTION_YAML
- HLF connection profile in YAML format
#### CHANNEL_TRANSFER_USERNAME
- Fabric user
#### CHANNEL_TRANSFER_USESMARTBFT
- Use SmartBFT consensus algorithm or Raft consensus algorithm


#### CHANNEL_TRANSFER_LISTENAPI_ACCESSTOKEN
- bearer token
#### CHANNEL_TRANSFER_LISTENAPI_ADDRESSHTTP
- the address and port for HTTP that will be open to receive connections
#### CHANNEL_TRANSFER_LISTENAPI_ADDRESSGRPC
- the address and port for GRPC that will be open to receive connections


#### CHANNEL_TRANSFER_REDISSTORAGE_ADDR
- redis addreses
#### CHANNEL_TRANSFER_REDISSTORAGE_PASSWORD
- redis auth
#### CHANNEL_TRANSFER_REDISSTORAGE_DBPREFIX
- redis key prefix
#### CHANNEL_TRANSFER_REDISSTORAGE_AFTERTRANSFERTTL
- time to life after transfer. Recorded set to 1 hour
#### CHANNEL_TRANSFER_REDISSTORAGE_CACERT
- CA Redis certificate in PEM encoding. The parameter must be filled when using the secret tls model
#### CHANNEL_TRANSFER_REDISSTORAGE_TLSHOSTNAMEOVERRIDE
- hostname override to use when checking TLS connection to Redis
#### CHANNEL_TRANSFER_REDISSTORAGE_CLIENTKEY
- secret key of the client. If mtls is disabled on the server side (tls-auth-clients no), then the parameter may be empty
#### CHANNEL_TRANSFER_REDISSTORAGE_CLIENTCERT
- client certificate. If mtls is disabled on the server side (tls-auth-clients no), then the parameter may be empty


#### CHANNEL_TRANSFER_SERVICE_ADDRESS
- web server port. Endpoints: / - liveness probe; /metrics - prometheus metrics; /ready - readiness probe



#### CHANNEL_TRANSFER_PROMMETRICS_PREFIX
- prometheus prefix without _
#### CHANNEL_TRANSFER_PROMMETRICS_ENABLEMETRICSREDIS
- turns metrics related to redis on


#### CHANNEL_TRANSFER_TRACING_ENABLEDTRACINGREDIS
- true turns redis calls tracing on
#### CHANNEL_TRANSFER_TRACING_COLLECTOR_ENDPOINT
- not empty value turns tracing on. it sets traces collector endpoint
#### CHANNEL_TRANSFER_TRACING_COLLECTOR_AUTHORIZATIONHEADERKEY
- traces collector authorization header key
#### CHANNEL_TRANSFER_TRACING_COLLECTOR_AUTHORIZATIONHEADERVALUE
- traces collector authorization header value
#### CHANNEL_TRANSFER_TRACING_COLLECTOR_TLSCA
- traces collector TLS CA certificate(s) in PEM encoding and then base64


#### CHANNEL_TRANSFER_CRYPTOSRC
- values: local, vault, google


#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTTOKEN
- access token for Vault
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_USERENEWABLEVAULTTOKENS
- use renewable vault tokens
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTADDRESS
- Vault instance address
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTAUTHPATH -
to which endpoint to send requests to get a temporary Vault token
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTROLE 
- Vault role
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTSERVICETOKENPATH 
- path to the token file for accessing the k8s
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_VAULTNAMESPACE 
- directory where all crypto materials are located in Vault
#### CHANNEL_TRANSFER_VAULTCRYPTOSETTINGS_USERCERT 
- Fabric user's cert


#### CHANNEL_TRANSFER_GOOGLECRYPTOSETTINGS_GCLOUDPROJECT
- GCloud project ID
#### CHANNEL_TRANSFER_GOOGLECRYPTOSETTINGS_GCLOUDCREDS
- path to GCloud credentials
#### CHANNEL_TRANSFER_GOOGLECRYPTOSETTINGS_USERCERT
- Fabric user's cert



#### CHANNEL_TRANSFER_CHANNELS
- comma separated ledger channel list of ATOMIZE for handle 

#### CHANNEL_TRANSFER_OPTIONS_BATCHTXPREIMAGEPREFIX
- transaction image prefix of batch world state composite key
#### CHANNEL_TRANSFER_OPTIONS_COLLECTORSBUFSIZE
- buffer size of blockData
#### CHANNEL_TRANSFER_OPTIONS_EXECUTETIMEOUT
- timeout of invoke HLF SDK method
#### CHANNEL_TRANSFER_OPTIONS_RETRYEXECUTEATTEMPTS
- number of attempts querying the HLF
#### CHANNEL_TRANSFER_OPTIONS_RETRYEXECUTEMAXDELAY
- delay between attempts querying the HLF
#### CHANNEL_TRANSFER_OPTIONS_WAITCOMMITATTEMPTS
- number of attempts checking that a transaction was committed in the HLF
#### CHANNEL_TRANSFER_OPTIONS_WAITCOMMITATTEMPTTIMEOUT
- timeout of attempts checking that a transaction was committed in the HLF
#### CHANNEL_TRANSFER_OPTIONS_TTL
- time to live of transfer. Recorded set to 3 hour
#### CHANNEL_TRANSFER_OPTIONS_TRANSFERSINHANDLEONCHANNEL
- the number of transfers currently being processed in the channel
#### CHANNEL_TRANSFER_OPTIONS_NEWESTREQUESTSTREAMBUFFERSIZE
- size of the request buffer of creating transfers

