package skills

// Skill 表示从 SKILL.md 解析出的可加载技能定义。
type Skill struct {
	Name         string
	Description  string
	Body         string
	Path         string
	Policy       Policy
	Dependencies Dependencies
}

type Policy struct {
	AllowImplicitInvocation bool
}

type Dependencies struct {
	Tools []string
}

type DiscoveryError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type DiscoveryResult struct {
	Skills []Skill
	Errors []DiscoveryError
}
