package hostharden

import (
	"fmt"
	"os"
	"strings"
)

// ImdsBlockUnitTemplatePath is the systemd template unit path used to create
// bridge-specific persistent instances (e.g., containarium-imds-block@br0.service).
// Using a template unit ensures that multiple bridges can have their own
// distinct persistent units without overwriting each other.
const ImdsBlockUnitTemplatePath = "/etc/systemd/system/containarium-imds-block@.service"

// InstallPersistentUnit writes and enables a systemd oneshot template instance that
// re-applies BlockMetadataFromBridge's rule for the specified bridge on every boot.
// Idempotent: re-running overwrites the content and `systemctl enable` is a no-op.
func InstallPersistentUnit(containariumBin, bridge string) error {
	// Use %i to represent the template instance parameter (the bridge name)
	unitPath := fmt.Sprintf("/etc/systemd/system/containarium-imds-block@%s.service", bridge)
	
	// Alternatively, we write/ensure the template file itself or instantiate directly.
	// Here we write to the template file /etc/systemd/system/containarium-imds-block@.service 
	// and enable the specific instance "containarium-imds-block@<bridge>.service".
	return installPersistentUnit(defaultRunner, ImdsBlockUnitTemplatePath, unitPath, containariumBin, bridge)
}

func installPersistentUnit(run runner, templatePath, unitPath, containariumBin, bridge string) error {
	// Define the systemd template unit content using %i for the bridge instance
	unit := fmt.Sprintf(`[Unit]
Description=Re-apply the BYOC metadata-endpoint FORWARD block (#1103) for bridge %i after reboot
After=network-online.target incus.socket
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s hostharden block-metadata %%i
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`, containariumBin)

	// 1. Write the template unit file (or instance-specific file, depending on design)
	// To follow systemd template standards, we write to the template path or unitPath.
	targetPath := "/etc/systemd/system/containarium-imds-block@.service"
	if err := os.WriteFile(targetPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", targetPath, err)
	}

	if out, err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// 2. Enable and start the specific instance for this bridge (e.g., containarium-imds-block@br0.service)
	instanceName := fmt.Sprintf("containarium-imds-block@%s.service", bridge)
	if out, err := run("systemctl", "enable", "--now", instanceName); err != nil {
		return fmt.Errorf("systemctl enable --now %s: %w: %s", instanceName, err, strings.TrimSpace(string(out)))
	}

	return nil
}
