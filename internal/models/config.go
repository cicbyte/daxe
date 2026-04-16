package models

type AppConfig struct {
	Version string `yaml:"version"` // 版本号，用于升级时判断
	AI      struct {
		Provider    string  `yaml:"provider"` // openai/ollama
		BaseURL     string  `yaml:"base_url"`
		ApiKey      string  `yaml:"api_key"`
		Model       string  `yaml:"model"`
		MaxTokens   int     `yaml:"max_tokens"`
		Temperature float64 `yaml:"temperature"`
		Timeout     int     `yaml:"timeout"`
	} `yaml:"ai"`

	Database struct {
		Type   string `yaml:"type"` // sqlite
		DbName string `yaml:"db_name"`
	} `yaml:"database"`

	Log struct {
		Level      string `yaml:"level"`
		MaxSize    int    `yaml:"maxSize"`
		MaxBackups int    `yaml:"maxBackups"`
		MaxAge     int    `yaml:"maxAge"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"log"`

	PicGo struct {
		Server string `yaml:"server"` // PicGo服务器地址
		Timeout int    `yaml:"timeout"` // 请求超时时间（秒）
	} `yaml:"picgo"`
}
