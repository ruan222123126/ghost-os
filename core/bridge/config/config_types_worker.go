package config

type WorkerConfig struct {
	Model          string
	MaxConcurrency int
	MaxFiles       int
	MaxFileChunks  int
}
