package transport

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
)

func runServeLegacyPreflight() error {
	findings := make([]string, 0, 8)

	promptFindings, err := bridgeconfig.DetectLegacyPromptState()
	if err != nil {
		return err
	}
	for _, finding := range promptFindings {
		findings = append(findings, fmt.Sprintf("%s; run `%s`", finding.Message, finding.MigrateCommand))
	}

	serverCfg, err := bridgeconfig.LoadServerConfig()
	if err != nil {
		return err
	}
	sessionStore, err := session.NewStore(serverCfg.SessionsPath)
	if err != nil {
		return err
	}
	defer sessionStore.Close()
	legacySessions, err := sessionStore.LegacySessionIDs()
	if err != nil {
		return err
	}
	if len(legacySessions) > 0 {
		findings = append(findings, fmt.Sprintf(
			"legacy session json files detected (%d): %s; run `bin/ghost-bridge migrate sessions`",
			len(legacySessions),
			strings.Join(legacySessions, ","),
		))
	}

	taskCfg, err := bridgeconfig.LoadTaskConfig()
	if err != nil {
		return err
	}
	legacyOrchestrations, err := bridgeorchestration.DetectLegacyOrchestrationTasks(taskCfg.TasksPath)
	if err != nil {
		return err
	}
	if len(legacyOrchestrations) > 0 {
		findings = append(findings, fmt.Sprintf(
			"orchestration tasks still contain removed start/end nodes (%d): %s; run `bin/ghost-bridge migrate orchestrations`",
			len(legacyOrchestrations),
			strings.Join(legacyOrchestrations, ","),
		))
	}

	if len(findings) == 0 {
		return nil
	}
	return fmt.Errorf("legacy compatibility cleanup required:\n- %s", strings.Join(findings, "\n- "))
}
