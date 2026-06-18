package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/mobile"
)

func runMobile(_ context.Context, args []string) (string, error) {
	if len(args) == 0 {
		return "", newUsageError("mobile requires a subcommand")
	}
	switch strings.TrimSpace(args[0]) {
	case "pair":
		return runMobilePair(args[1:])
	case "devices":
		if len(args) != 1 {
			return "", newUsageError("mobile devices does not accept arguments")
		}
		return runMobileDevices()
	case "revoke":
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return "", newUsageError("mobile revoke requires <device_id>")
		}
		return runMobileRevoke(args[1])
	default:
		return "", newUsageError(fmt.Sprintf("unknown mobile subcommand %q", args[0]))
	}
}

func runMobilePair(args []string) (string, error) {
	if len(args) > 1 {
		return "", newUsageError("mobile pair accepts at most one label")
	}
	label := "mobile"
	if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
		label = strings.TrimSpace(args[0])
	}
	cfg, err := bridgeconfig.LoadMobileWebRTCConfig()
	if err != nil {
		return "", err
	}
	iceJSON, err := mobileICEServersJSON(cfg.ICEServers)
	if err != nil {
		return "", err
	}
	result, err := mobile.NewCredentialStore(cfg.CredentialStorePath).Pair(mobile.PairingOptions{
		Label:          label,
		PCID:           cfg.PCID,
		SignalingURL:   cfg.SignalingURL,
		SignalingToken: cfg.SignalingToken,
		ICEServersJSON: iceJSON,
		Now:            time.Now(),
	})
	if err != nil {
		return "", err
	}
	return formatPairingOutput(result), nil
}

func runMobileDevices() (string, error) {
	cfg, err := bridgeconfig.LoadMobileWebRTCConfig()
	if err != nil {
		return "", err
	}
	devices, err := mobile.NewCredentialStore(cfg.CredentialStorePath).List()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "no paired mobile devices", nil
	}
	var b strings.Builder
	b.WriteString("device_id\tlabel\tstatus\tlast_connected_at\n")
	for _, device := range devices {
		lastConnected := ""
		if device.LastConnectedAt != nil {
			lastConnected = device.LastConnectedAt.Format(time.RFC3339)
		}
		status := "active"
		if device.Revoked {
			status = "revoked"
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", device.DeviceID, device.Label, status, lastConnected)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func runMobileRevoke(deviceID string) (string, error) {
	cfg, err := bridgeconfig.LoadMobileWebRTCConfig()
	if err != nil {
		return "", err
	}
	if err := mobile.NewCredentialStore(cfg.CredentialStorePath).Revoke(deviceID); err != nil {
		return "", err
	}
	return fmt.Sprintf("revoked mobile device %s", strings.TrimSpace(deviceID)), nil
}

func mobileICEServersJSON(servers []bridgeconfig.MobileICEServerConfig) (string, error) {
	if len(servers) == 0 {
		return "", nil
	}
	data, err := json.Marshal(servers)
	if err != nil {
		return "", fmt.Errorf("encode mobile ice servers: %w", err)
	}
	return string(data), nil
}

func formatPairingOutput(result mobile.PairingResult) string {
	var qr bytes.Buffer
	qrterminal.GenerateHalfBlock(result.URI, qrterminal.L, &qr)

	var b strings.Builder
	fmt.Fprintf(&b, "device_id: %s\n", result.Device.DeviceID)
	fmt.Fprintf(&b, "label: %s\n", result.Device.Label)
	b.WriteString("pairing_uri:\n")
	b.WriteString(result.URI)
	b.WriteString("\nqr:\n")
	b.WriteString(qr.String())
	return strings.TrimRight(b.String(), "\n")
}
