package config

type Grafana struct {
	config ProductConfig
}

func NewGrafana(config ProductConfig) *Grafana {
	return &Grafana{config: config}
}

func (s *Grafana) GetNamespace() string {
	return s.config["NAMESPACE"]
}

func (s *Grafana) SetNamespace(newNamespace string) {
	s.config["NAMESPACE"] = newNamespace
}

func (s *Grafana) GetOperatorNamespace() string {
	return s.config["OPERATOR_NAMESPACE"]
}

func (s *Grafana) SetOperatorNamespace(newNamespace string) {
	s.config["OPERATOR_NAMESPACE"] = newNamespace
}

func (s *Grafana) Read() ProductConfig {
	return s.config
}

func (s *Grafana) GetHost() string {
	return s.config["HOST"]
}

func (s *Grafana) GetLabelSelector() string {
	return "middleware"
}

func (s *Grafana) SetHost(newHost string) {
	s.config["HOST"] = newHost
}
