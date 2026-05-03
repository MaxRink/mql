// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package networki

import (
	"bufio"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

// detectAIXInterfaces detects network interfaces on AIX. AIX is in the
// FAMILY_UNIX family but not FAMILY_BSD or FAMILY_LINUX, so dispatch is
// keyed on Platform.Name == "aix".
//
// AIX is special among unixes: `ifconfig -a` does not emit the link-layer
// address or the MTU on the header line, so a second pass through
// `netstat -in` is required to fill those fields.
func (n *neti) detectAIXInterfaces() ([]Interface, error) {
	detectors := []func() ([]Interface, error){
		n.getAIXIfconfigInterfaces,
		n.getAIXNetstatInInterfaces,
		n.getAIXGatewayDetails,
	}

	var errs []error
	interfaces := []Interface{}
	for _, fn := range detectors {
		detected, err := fn()
		if err != nil {
			log.Debug().Err(err).Msg("os.network.interface> unable to detect network interfaces")
			errs = append(errs, err)
			continue
		}
		interfaces = AddOrUpdateInterfaces(interfaces, detected)
	}
	if len(interfaces) == 0 {
		return interfaces, errors.Join(errs...)
	}
	return interfaces, nil
}

// aixIfconfigHeaderRegex tolerates AIX's "<hex>" and "<hex>,<hex>" flag
// prefixes and a bracketed name list that may itself contain parens
// (e.g. "CHECKSUM_OFFLOAD(ACTIVE)").
var aixIfconfigHeaderRegex = regexp.MustCompile(`^([a-zA-Z0-9._]+):\s+flags=[^<]*<([^>]*)>`)

func (n *neti) getAIXIfconfigInterfaces() (interfaces []Interface, err error) {
	output, err := n.RunCommand("ifconfig -a")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var current *Interface
	for scanner.Scan() {
		raw := scanner.Text()
		if raw == "" {
			continue
		}

		if m := aixIfconfigHeaderRegex.FindStringSubmatch(raw); len(m) > 0 {
			if current != nil {
				interfaces = append(interfaces, *current)
			}
			current = &Interface{Name: m[1]}
			if m[2] != "" {
				for _, f := range strings.Split(m[2], ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						current.Flags = append(current.Flags, f)
					}
				}
			}
			// AIX has no `status:` line; derive Active from the UP flag.
			for _, f := range current.Flags {
				if f == "UP" {
					current.Active = convert.ToPtr(true)
					break
				}
			}
			if current.Active == nil {
				current.Active = convert.ToPtr(false)
			}
			continue
		}

		if current == nil {
			continue
		}

		fields := strings.Fields(strings.TrimSpace(raw))
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "inet":
			ipField := fields[1]
			if slash := strings.Index(ipField, "/"); slash != -1 {
				current.AddOrUpdateIP(NewIPv4WithPrefixLength(
					ipField[:slash], parseInt(ipField[slash+1:]),
				))
			} else if len(fields) >= 4 && fields[2] == "netmask" {
				current.AddOrUpdateIP(NewIPv4WithMask(ipField, fields[3]))
			} else if ip, ok := NewIPAddress(ipField); ok {
				current.AddOrUpdateIP(ip)
			}

		case "inet6":
			// AIX form: "inet6 ::1%1/0" or "inet6 fe80::1/64".
			ipField := fields[1]
			prefix := -1
			if slash := strings.Index(ipField, "/"); slash != -1 {
				prefix = parseInt(ipField[slash+1:])
				ipField = ipField[:slash]
			}
			if pct := strings.Index(ipField, "%"); pct != -1 {
				ipField = ipField[:pct]
			}
			if prefix >= 0 {
				current.AddOrUpdateIP(NewIPv6WithPrefixLength(ipField, prefix))
			} else if ip, ok := NewIPAddress(ipField); ok {
				current.AddOrUpdateIP(ip)
			}
		}
	}

	if current != nil {
		interfaces = append(interfaces, *current)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.ifconfig").
		Msg("os.network.interfaces> discovered")
	return interfaces, nil
}

// aixDottedMACRegex matches AIX's `netstat -in` MAC format: dotted hex
// octets with leading zeros stripped, e.g. "0.50.56.b0.9a.a5".
var aixDottedMACRegex = regexp.MustCompile(`^[0-9a-fA-F]{1,2}(?:\.[0-9a-fA-F]{1,2}){5}$`)

func aixNormalizeMAC(s string) string {
	if !aixDottedMACRegex.MatchString(s) {
		return ""
	}
	parts := strings.Split(s, ".")
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return strings.ToLower(strings.Join(parts, ":"))
}

// getAIXNetstatInInterfaces extracts MTU and MAC from `netstat -in`.
// Only the "link#N" rows are read; per-IP rows duplicate data already
// captured by ifconfig.
func (n *neti) getAIXNetstatInInterfaces() (interfaces []Interface, err error) {
	output, err := n.RunCommand("netstat -in")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Name") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if !strings.HasPrefix(fields[2], "link#") {
			continue
		}
		iface := Interface{
			Name: fields[0],
			MTU:  parseInt(fields[1]),
		}
		// MAC is absent for pseudo-interfaces (e.g. lo0); SetMAC handles
		// the empty-string no-op for us.
		if len(fields) >= 4 {
			iface.SetMAC(aixNormalizeMAC(fields[3]))
		}
		interfaces = append(interfaces, iface)
	}

	log.Debug().
		Interface("interfaces", interfaces).
		Str("detector", "cmd.netstat_in").
		Msg("os.network.interfaces> discovered")
	return
}

// getAIXGatewayDetails extracts default-gateway info from `netstat -rn`.
// AIX prints two route trees ("Internet" then "Internet v6"); the
// default rows in each contribute the v4 and v6 gateways respectively.
func (n *neti) getAIXGatewayDetails() (interfaces []Interface, err error) {
	output, err := n.RunCommand("netstat -rn")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) < 4 || !isDefaultRoute(fields[0]) {
			continue
		}
		gateway := fields[1]
		if pct := strings.Index(gateway, "%"); pct != -1 {
			gateway = gateway[:pct]
		}
		version := IPv4
		if strings.Contains(gateway, ":") {
			version = IPv6
		}
		netif := ""
		for i := len(fields) - 1; i >= 2; i-- {
			if isLikelyAIXIfname(fields[i]) {
				netif = fields[i]
				break
			}
		}
		if netif == "" {
			continue
		}

		gw := gateway
		ver := version
		interfaces = append(interfaces, Interface{
			Name: netif,
			enrichments: func(in *Interface) {
				for i := range in.IPAddresses {
					v, ok := in.IPAddresses[i].Version()
					if !ok || v != ver {
						continue
					}
					in.IPAddresses[i].Gateway = gw
				}
			},
		})
	}
	return
}

// aixIfnameRegex matches typical AIX interface names: en0, en1, lo0,
// et0, fcs0, sit0. AIX names always end in a digit run with no dotted
// suffix.
var aixIfnameRegex = regexp.MustCompile(`^[a-zA-Z]+\d+$`)

func isLikelyAIXIfname(s string) bool {
	return aixIfnameRegex.MatchString(s)
}
