package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/footprintai/containarium/internal/hostharden"
)

var hostHardenCmd = &cobra.Command{
	Use:    "hostharden",
	Short:  "Narrow host-hardening mutations, applied by cloud enroll / pool join and the reboot unit they install",
	Hidden: true,
}

var (
	persistBlockMetadata bool

	hostHardenBlockMetadataCmd = &cobra.Command{
		Use:   "block-metadata <bridge>",
		Short: "Idempotently drop FORWARDED traffic from <bridge>'s subnet to the cloud metadata endpoint",
		Long: `Inserts (if not already present) an iptables FORWARD rule dropping traffic
from <bridge>'s configured subnet to 169.254.169.254 — the cloud metadata
endpoint every major provider serves at that link-local address. Scoped to
FORWARDED (container-bridge) traffic only; the host's own OUTPUT-originated
requests are untouched, so cloud-provider tooling running on the host itself
keeps working. See internal/hostharden and #1103.`,
		Args: cobra.ExactArgs(1),
		RunE: runHostHardenBlockMetadata,
	}
)

func init() {
	rootCmd.AddCommand(hostHardenCmd)
	hostHardenCmd.AddCommand(hostHardenBlockMetadataCmd)
	hostHardenBlockMetadataCmd.Flags().BoolVar(&persistBlockMetadata, "persist", false, "Install and enable boot-time re-apply systemd unit")
}

// runHostHardenBlockMetadata executes the block-metadata command for a given bridge,
// applying the firewall rule and optionally installing a persistent systemd unit
// if the --persist flag is provided.
func runHostHardenBlockMetadata(cmd *cobra.Command, args []string) error {
	bridge := args[0]
	applied, detail, err := hostharden.BlockMetadataFromBridge(bridge)
	if err != nil {
		return err
	}
	
	mark := " "
	if applied {
		mark = "✓"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", mark, detail)

	if persistBlockMetadata {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine containarium executable path: %w", err)
		}
		if err := hostharden.InstallPersistentUnit(execPath, bridge); err != nil {
			return fmt.Errorf("failed to install persistent unit: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ persistent systemd unit installed and enabled for bridge %s\n", bridge)
	}

	return nil
}
