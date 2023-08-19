package entities

type IntegrationKNoTConfig struct {
	UserToken               string `yaml:"user_token"`
	URL                     string `yaml:"url"`
	EventRoutingKeyTemplate string `yaml:"event_routing_key_template"`
}
