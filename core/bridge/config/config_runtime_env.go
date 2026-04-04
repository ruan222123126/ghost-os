package config

func getenvDefault(name, fallback string) string {
	return currentEnv().defaultValue(name, fallback)
}
