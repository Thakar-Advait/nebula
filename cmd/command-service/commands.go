package commandservice

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

//[DNS] add modifyDNSWindows()
func modifyDNSWindows(interfaceAlias string, dnsAddr string) (*exec.Cmd, error) {
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias "%s" -ServerAddresses "%s"`, interfaceAlias, dnsAddr))
	return cmd, nil
}

//[DNS] add modifyDNSLinux 
func modifyDNSLinux(interfaceAlias string, dnsAddr string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("resolvectl"); err == nil {
		return exec.Command("resolvectl", "dns", interfaceAlias, dnsAddr), nil
	}

	if _, err := exec.LookPath("nmcli"); err == nil {
		return exec.Command("nmcli", "connection", "modify", interfaceAlias,
			"ipv4.dns", dnsAddr, "ipv4.ignore-auto-dns", "yes"), nil
	}

	return nil, fmt.Errorf("no supported DNS manager found (resolvectl or nmcli)")
}

//[DNS] add modifyDNSMac
func modifyDNSMac(_ string, dnsAddr string) (*exec.Cmd, error) {
	cmd := exec.Command("networksetup", "-setdnsservers", "Wi-Fi", dnsAddr)
	return cmd, nil
}

//[DNS] add flushDNSWindows
func flushDNSWindows() (*exec.Cmd, error) {
	return exec.Command("powershell", "-Command", "Clear-DnsClientCache"), nil
}

//[DNS] add flushDNSLinux
func flushDNSLinux() (*exec.Cmd, error) {
	if _, err := exec.LookPath("resolvectl"); err == nil {
		return exec.Command("resolvectl", "flush-caches"), nil
	}
	if _, err := exec.LookPath("systemd-resolve"); err == nil {
		return exec.Command("systemd-resolve", "--flush-caches"), nil
	}
	return nil, fmt.Errorf("no supported DNS flush tool found (resolvectl or systemd-resolve)")
}

//[DNS] add flushDNSMac
func flushDNSMac() (*exec.Cmd, error) {
	return exec.Command("sh", "-c", "sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder"), nil
}

//[DNS] add restartNetworkAdapter
func restartNetworkAdapterWindows(interfaceAlias string) (*exec.Cmd, error) {
	return exec.Command("powershell", "-Command",
		fmt.Sprintf(`Restart-NetAdapter -Name "%s" -Confirm:$false`, interfaceAlias)), nil
}

//[DNS] add restartNetworkAdapter
func restartNetworkAdapterLinux(interfaceAlias string) (*exec.Cmd, error) {
	return exec.Command("sh", "-c",
		fmt.Sprintf("ip link set %s down && ip link set %s up", interfaceAlias, interfaceAlias)), nil
}

//[DNS] add restartNetworkAdapter
func restartNetworkAdapterMac(_ string) (*exec.Cmd, error) {
	return exec.Command("sh", "-c", "networksetup -setdnsservers Wi-Fi empty"), nil
}

//[DNS] add SetDns cross platform
func SetDns(interfaceAlias string, dnsAddr string) error {
	var cmd *exec.Cmd
	var err error

	switch runtime.GOOS {
	case "windows":
		cmd, err = modifyDNSWindows(interfaceAlias, dnsAddr)
	case "linux":
		cmd, err = modifyDNSLinux(interfaceAlias, dnsAddr)
	case "darwin":
		cmd, err = modifyDNSMac(interfaceAlias, dnsAddr)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("failed to prepare DNS command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("DNS modification failed: %v\nOutput: %s", err, output)
	}

	if runtime.GOOS == "linux" {
		applyCmd := exec.Command("nmcli", "connection", "up", interfaceAlias)
		out, err := applyCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to bring Linux connection up: %v\nOutput: %s", err, out)
		}
	}

	return nil
}

//[DNS] add FlushDNS cross platform
func FlushDNS() error {
	var cmd *exec.Cmd
	var err error

	switch runtime.GOOS {
	case "windows":
		cmd, err = flushDNSWindows()
	case "linux":
		cmd, err = flushDNSLinux()
	case "darwin":
		cmd, err = flushDNSMac()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("failed to prepare flush DNS command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("DNS flush failed: %v\nOutput: %s", err, output)
	}

	return nil
}

//[DNS] add revertDns cross platform 
func ResetDns(interfaceAlias string) error {
	var flushCmd *exec.Cmd
	var restartCmd *exec.Cmd
	var err error

	// Flush DNS
	switch runtime.GOOS {
	case "windows":
		flushCmd, err = flushDNSWindows()
	case "linux":
		flushCmd, err = flushDNSLinux()
	case "darwin":
		flushCmd, err = flushDNSMac()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("failed to prepare flush DNS command: %v", err)
	}

	flushOutput, err := flushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("DNS flush failed: %v\nOutput: %s", err, flushOutput)
	}

	// Restart network adapter
	switch runtime.GOOS {
	case "windows":
		restartCmd, err = restartNetworkAdapterWindows(interfaceAlias)
	case "linux":
		restartCmd, err = restartNetworkAdapterLinux(interfaceAlias)
	case "darwin":
		restartCmd, err = restartNetworkAdapterMac(interfaceAlias)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return fmt.Errorf("failed to prepare network restart command: %v", err)
	}

	restartOutput, err := restartCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("network adapter restart failed: %v\nOutput: %s", err, restartOutput)
	}

	return nil
}

func CommandToString(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func getDefaultGatewayCommandWindows() (*exec.Cmd, error) {
	cmd := exec.Command("powershell.exe", "-Command",
		`Get-NetRoute -DestinationPrefix "0.0.0.0/0" | Where-Object { $_.NextHop -ne '0.0.0.0' -and $_.NextHop -ne '::' } | Sort-Object { $_.RouteMetric + $_.InterfaceMetric } | Select-Object -ExpandProperty NextHop -First 1`)
	return cmd, nil
}

func getDefaultGatewayCommandLinux() (*exec.Cmd, error) {
	panic("not implemented")
}

func getDefaultGatewayCommandMac() (*exec.Cmd, error) {
	return exec.Command("sh", "-c", "route -n get default | grep 'gateway' | awk '{print $2}'"), nil
}

func FetchDefaultGateway() (net.IP, error) {
	var cmd *exec.Cmd
	var err error

	switch runtime.GOOS {
	case "windows":
		cmd, err = getDefaultGatewayCommandWindows()
	case "linux":
		cmd, err = getDefaultGatewayCommandLinux()
	case "darwin":
		cmd, err = getDefaultGatewayCommandMac()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to prepare default gateway command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get default gateway: %v\nOutput: %s", err, output)
	}

	return net.ParseIP(strings.TrimSpace(string(output))), nil
}

func getDefaultNetworkInterfaceWindows() (*exec.Cmd, error) {
	cmd := exec.Command("powershell", "-Command",
		`Get-NetRoute -DestinationPrefix "0.0.0.0/0" | 
          Where-Object { $_.NextHop -ne '0.0.0.0' -and $_.NextHop -ne '::' } | 
          Sort-Object { $_.RouteMetric + $_.InterfaceMetric } | 
          Select-Object -ExpandProperty InterfaceAlias -First 1`)
	return cmd, nil
}

func getDefaultNetworkInterfaceLinux() (*exec.Cmd, error) {
	cmd := exec.Command("sh", "-c", `ip route get 8.8.8.8 | grep -oP 'dev \K\w+'`)
	return cmd, nil
}

func getDefaultNetworkInterfaceMac() (*exec.Cmd, error) {
	return exec.Command("sh", "-c", "route -n get default | grep interface | awk '{print $2}'"), nil
}

func GetDefaultNetworkInterface() (string, error) {
	var cmd *exec.Cmd
	var err error
	switch runtime.GOOS {
	case "windows":
		cmd, err = getDefaultNetworkInterfaceWindows()
	case "linux":
		cmd, err = getDefaultNetworkInterfaceLinux()
	case "darwin":
		cmd, err = getDefaultNetworkInterfaceMac()
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err != nil {
		return "", fmt.Errorf("failed to prepare default network interface command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get default network interface: %v\nOutput: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

func getAddRouteCommandWindows(dest net.IP, via net.IP) (*exec.Cmd, error) {
	if dest == nil || via == nil {
		return nil, fmt.Errorf("destination or gateway cannot be nil")
	}
	psCommand := fmt.Sprintf(`route add %s mask 255.255.255.255 %s`, dest, via)

	return exec.Command("powershell", "-Command", psCommand), nil
}

func getAddRouteCommandLinux(dest net.IP, via net.IP) (*exec.Cmd, error) {
	if dest == nil || via == nil {
		return nil, fmt.Errorf("destination or gateway cannot be nil")
	}

	return exec.Command("sh", "-c", fmt.Sprintf("ip route add %s via %s", dest, via)), nil
}

func getAddRouteCommandMac(dest net.IP, via net.IP) (*exec.Cmd, error) {
	if dest == nil || via == nil {
		return nil, fmt.Errorf("destination or gateway cannot be nil")
	}

	return exec.Command("sudo", "route", "add", "-host", dest.String(), via.String()), nil
}

func getAddDefaultRouteCommandWindows(via net.IP) (*exec.Cmd, error) {
	panic("not implemented")
}

func getAddDefaultRouteCommandLinux(via net.IP) (*exec.Cmd, error) {
	panic("not implemented")
}

func getAddDefaultRouteCommandMac(via net.IP) (*exec.Cmd, error) {
	return exec.Command("sudo", "route", "add", "default", via.String()), nil
}

func AddDefaultRoute(via net.IP) error {
	var cmd *exec.Cmd
	var err error

	switch runtime.GOOS {
	case "windows":
		cmd, err = getAddDefaultRouteCommandWindows(via)
	case "linux":
		cmd, err = getAddDefaultRouteCommandLinux(via)
	case "darwin":
		cmd, err = getAddDefaultRouteCommandMac(via)
	default:
		return fmt.Errorf("")
	}
	if err != nil {
		return fmt.Errorf("failed to prepare add default route command: %v", err)
	}
	fmt.Println("[DEV] Executing command: ", CommandToString(cmd))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add default route: %v\nOutput: %s", err, output)
	}
	return nil
}

func AddFullTunnelRoute(dest net.IP, via net.IP) error {
	if dest == nil || via == nil {
		return fmt.Errorf("destination or gateway cannot be nil")
	}

	var cmd *exec.Cmd
	var err error

	switch runtime.GOOS {
	case "windows":
		cmd, err = getAddRouteCommandWindows(dest, via)
	case "linux":
		cmd, err = getAddRouteCommandLinux(dest, via)
	case "darwin":
		cmd, err = getAddRouteCommandMac(dest, via)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	fmt.Println("[DEV] Executing command: ", CommandToString(cmd))
	if err != nil {
		return fmt.Errorf("failed to prepare add host route command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add host route: %v\nOutput: %s", err, output)
	}

	return nil
}

func getDeleteRouteCommandWindows(dest net.IP) (*exec.Cmd, error) {
	if dest == nil {
		return nil, fmt.Errorf("destination cannot be nil")
	}
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("route delete %s mask 255.255.255.255", dest.String()),
	)
	return cmd, nil
}

func getDeleteRouteCommandLinux(dest net.IP, via net.IP) (*exec.Cmd, error) {
	if dest == nil {
		return nil, fmt.Errorf("destination cannot be nil")
	}

	return exec.Command("sudo", "ip", "route", "del", dest.String(), "via", via.String()), nil
}

func getDeleteRouteCommandMac(dest net.IP) (*exec.Cmd, error) {
	if dest == nil {
		return nil, fmt.Errorf("destination cannot be nil")
	}

	return exec.Command("sudo", "route", "-n", "delete", "-host", dest.String()), nil
}

func DeleteFullTunnelRoute(dest net.IP, via net.IP) error {
	if dest == nil {
		return fmt.Errorf("destination cannot be nil")
	}

	var cmd *exec.Cmd
	var err error
	switch runtime.GOOS {
	case "windows":
		cmd, err = getDeleteRouteCommandWindows(dest)
	case "linux":
		cmd, err = getDeleteRouteCommandLinux(dest, via)
	case "darwin":
		cmd, err = getDeleteRouteCommandMac(dest)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	if err != nil {
		return fmt.Errorf("failed to prepare delete host route command: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete host route: %v\nOutput: %s", err, output)
	}

	return nil
}
