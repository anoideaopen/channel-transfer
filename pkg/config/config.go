package config

import (
	"crypto/tls"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

const (
	EnvPrefix             = "CHANNEL_TRANSFER"
	sensitiveDataMask     = "****"
	defaultConfigFilePath = "config.yaml"
)

type Config struct {
	LogLevel    string `mapstructure:"logLevel" validate:"required"`
	LogType     string `mapstructure:"logType" validate:"required"`
	ProfilePath string `mapstructure:"profilePath" validate:"required"`
	UserName    string `mapstructure:"userName" validate:"required"`

	ListenAPI    *ListenAPI    `mapstructure:"listenAPI"`
	RedisStorage *RedisStorage `mapstructure:"redisStorage" validate:"required"`
	Service      *Service      `mapstructure:"service" validate:"required"`
	PromMetrics  *PromMetrics  `mapstructure:"promMetrics"`

	Channels []string `mapstructure:"channels" validate:"required"`

	Options Options `mapstructure:"options" validate:"dive"`
}

type ListenAPI struct {
	AccessToken string `mapstructure:"accessToken"`
	AddressHTTP string `mapstructure:"addressHTTP" validate:"required"`
	AddressGRPC string `mapstructure:"addressGRPC" validate:"required"`

	tlsConfig *tls.Config `mapstructure:"-"`
}

func (api *ListenAPI) TLSConfig() *tls.Config {
	return api.tlsConfig
}

type PromMetrics struct {
	PrefixForMetrics string `mapstructure:"prefix"`
}

type Options struct {
	BatchTxPreimagePrefix         string         `mapstructure:"batchTxPreimagePrefix"  validate:"required"`
	CollectorsBufSize             uint           `mapstructure:"collectorsBufSize" validate:"required"`
	ExecuteTimeout                *time.Duration `mapstructure:"executeTimeout"`
	RetryExecuteAttempts          uint           `mapstructure:"retryExecuteAttempts" validate:"required"`
	RetryExecuteMaxDelay          *time.Duration `mapstructure:"retryExecuteMaxDelay"`
	RetryExecuteDelay             *time.Duration `mapstructure:"retryExecuteDelay"`
	TTL                           *time.Duration `mapstructure:"ttl"`
	TransfersInHandleOnChannel    uint           `mapstructure:"transfersInHandleOnChannel" validate:"required"`
	NewestRequestStreamBufferSize uint           `mapstructure:"newestRequestStreamBufferSize" validate:"required"`
}

func (eo Options) EffTTL(defOpts Options) (time.Duration, error) {
	var val *time.Duration
	if eo.TTL != nil {
		val = eo.TTL
	} else if defOpts.TTL != nil {
		val = defOpts.TTL
	}
	if val == nil {
		return 0, errors.New("TTL is not specified")
	}
	return *val, nil
}

func (eo Options) EffExecuteTimeout(defOpts Options) (time.Duration, error) {
	var val *time.Duration
	if eo.ExecuteTimeout != nil {
		val = eo.ExecuteTimeout
	} else if defOpts.ExecuteTimeout != nil {
		val = defOpts.ExecuteTimeout
	}
	if val == nil {
		return 0, errors.New("executeTimeout is not specified")
	}
	return *val, nil
}

func (eo Options) EffRetryExecuteMaxDelay(defOpts Options) (time.Duration, error) {
	var val *time.Duration
	if eo.RetryExecuteMaxDelay != nil {
		val = eo.RetryExecuteMaxDelay
	} else if defOpts.RetryExecuteMaxDelay != nil {
		val = defOpts.RetryExecuteMaxDelay
	}
	if val == nil {
		return 0, errors.New("retryExecuteMaxDelay is not specified")
	}
	return *val, nil
}

func (eo Options) EffRetryExecuteDelay(defOpts Options) (time.Duration, error) {
	var val *time.Duration
	if eo.RetryExecuteDelay != nil {
		val = eo.RetryExecuteDelay
	} else if defOpts.RetryExecuteDelay != nil {
		val = defOpts.RetryExecuteDelay
	}
	if val == nil {
		return 0, errors.New("retryExecuteDelay is not specified")
	}
	return *val, nil
}

type RedisStorage struct {
	Addr                []string       `mapstructure:"addr" validate:"required"`
	Password            string         `mapstructure:"password"`
	DBPrefix            string         `mapstructure:"dbPrefix" validate:"required"`
	AfterTransferTTL    *time.Duration `mapstructure:"afterTransferTTL" validate:"required"`
	CACert              string         `mapstructure:"caCert,omitempty"`
	TLSHostnameOverride string         `mapstructure:"tlsHostnameOverride,omitempty"`
	ClientKey           string         `mapstructure:"clientKey,omitempty"`
	ClientCert          string         `mapstructure:"clientCert,omitempty"`

	tlsConfig *tls.Config `mapstructure:"-"`
}

func (rs *RedisStorage) TLSConfig() *tls.Config {
	return rs.tlsConfig
}

func (rs *RedisStorage) EffAfterTransferTTL(cfg RedisStorage) (time.Duration, error) {
	var val *time.Duration
	if rs.AfterTransferTTL != nil {
		val = rs.AfterTransferTTL
	} else if cfg.AfterTransferTTL != nil {
		val = cfg.AfterTransferTTL
	}
	if val == nil {
		return 0, errors.New("AfterTransferTTL is not specified")
	}
	return *val, nil
}

func GetConfig() (*Config, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}

	if err = validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateConfig(cfg *Config) error {
	err := validateRequiredFields(cfg)
	if err != nil {
		return err
	}

	if len(cfg.Channels) == 0 {
		return errors.New("channel not found")
	}

	if cfg.Options.CollectorsBufSize == 0 {
		return errors.New("collectorsBufSize must be positive")
	}
	if _, err = cfg.Options.EffExecuteTimeout(cfg.Options); err != nil {
		return errors.New(err)
	}
	if _, err = cfg.Options.EffTTL(cfg.Options); err != nil {
		return errors.New(err)
	}
	if _, err = cfg.Options.EffRetryExecuteMaxDelay(cfg.Options); err != nil {
		return errors.New(err)
	}
	if _, err = cfg.Options.EffRetryExecuteDelay(cfg.Options); err != nil {
		return errors.New(err)
	}

	if _, err = cfg.RedisStorage.EffAfterTransferTTL(*cfg.RedisStorage); err != nil {
		return errors.New(err)
	}

	return nil
}

func validateRequiredFields(cfg *Config) error {
	validate := validator.New()
	err := validate.Struct(*cfg)
	if err != nil {
		return errors.Errorf("err(s):\n%+v", err)
	}
	return nil
}

func setConfigFile() {
	// 1. params
	if p, ok := getConfigPathFromParams(); ok {
		viper.SetConfigFile(p)
		return
	}
	// 2. env
	if p, ok := os.LookupEnv(EnvPrefix + "_CONFIG"); ok {
		viper.SetConfigFile(p)
		return
	}
	// 3. default
	viper.SetConfigFile(defaultConfigFilePath)
	viper.AddConfigPath("/etc")
}

func getConfig() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvPrefix(EnvPrefix)

	setConfigFile()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Errorf("failed viper.ReadInConfig: %w", err)
	}

	cfg := Config{}
	if err := viper.UnmarshalExact(&cfg); err != nil {
		return nil, errors.Errorf("failed viper.Unmarshal: %w", err)
	}

	return &cfg, nil
}

func getConfigPathFromParams() (string, bool) {
	flg := flag.Lookup("c")
	if flg != nil {
		return flg.Value.String(), true
	}

	p := flag.String(
		"c",
		"",
		"Configuration file path",
	)
	flag.Parse()

	return *p, *p != ""
}

// WithoutSensitiveData returns copy of config with empty sensitive data. This config might be used for trace logging.
func (c Config) WithoutSensitiveData() Config {
	return Config{
		LogLevel:     c.LogLevel,
		LogType:      c.LogType,
		ProfilePath:  c.ProfilePath,
		UserName:     c.UserName,
		RedisStorage: c.RedisStorage.withoutSensitiveData(),
		PromMetrics:  c.PromMetrics,
		Channels:     c.Channels,
		Options:      c.Options.withoutSensitiveData(),
	}
}

func (eo Options) withoutSensitiveData() Options {
	return Options{
		ExecuteTimeout:                eo.ExecuteTimeout,
		TTL:                           eo.TTL,
		TransfersInHandleOnChannel:    eo.TransfersInHandleOnChannel,
		NewestRequestStreamBufferSize: eo.NewestRequestStreamBufferSize,
	}
}

func (rs *RedisStorage) withoutSensitiveData() *RedisStorage {
	if rs == nil {
		return nil
	}
	return &RedisStorage{
		Addr:                []string{sensitiveDataMask},
		Password:            sensitiveDataMask,
		DBPrefix:            rs.DBPrefix,
		CACert:              sensitiveDataMask,
		TLSHostnameOverride: sensitiveDataMask,
		ClientCert:          sensitiveDataMask,
		ClientKey:           sensitiveDataMask,
	}
}

type Service struct {
	Address string `mapstructure:"address" validate:"required"`

	tlsConfig *tls.Config `mapstructure:"-"`
}

func (api *Service) TLSConfig() *tls.Config {
	return api.tlsConfig
}
