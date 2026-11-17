package main

import (
	"bufio"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func loadList(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	res := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		res[strings.ToUpper(line)] = true
	}
	return res, sc.Err()
}

func parseClassicalYAML(path string) ([]*routercommon.Domain, []*routercommon.CIDR, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	var domains []*routercommon.Domain
	var cidrs []*routercommon.CIDR

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		item = strings.Trim(item, "\"'")
		parts := strings.Split(item, ",")
		if len(parts) < 2 {
			continue
		}
		kind := strings.ToUpper(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch kind {
		case "DOMAIN":
			domains = append(domains, &routercommon.Domain{
				Type:  routercommon.Domain_Full,
				Value: val,
			})
		case "DOMAIN-SUFFIX":
			domains = append(domains, &routercommon.Domain{
				Type:  routercommon.Domain_RootDomain,
				Value: val,
			})
		case "DOMAIN-KEYWORD":
			domains = append(domains, &routercommon.Domain{
				Type:  routercommon.Domain_Plain,
				Value: val,
			})
		case "IP-CIDR", "IP-CIDR6":
			pfx, err := netip.ParsePrefix(val)
			if err != nil {
				continue
			}
			pfx = pfx.Masked()
			addr := pfx.Addr()
			cidrs = append(cidrs, &routercommon.CIDR{
				Ip:     addr.AsSlice(),
				Prefix: uint32(pfx.Bits()),
			})
		}
	}
	return domains, cidrs, sc.Err()
}

func dedupDomains(domains []*routercommon.Domain) []*routercommon.Domain {
	seen := make(map[string]bool, len(domains))
	out := make([]*routercommon.Domain, 0, len(domains))
	for _, d := range domains {
		key := fmt.Sprintf("%d:%s", d.Type, strings.ToLower(d.Value))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	return out
}

func dedupCIDRs(cidrs []*routercommon.CIDR) []*routercommon.CIDR {
	seen := make(map[string]bool, len(cidrs))
	out := make([]*routercommon.CIDR, 0, len(cidrs))
	for _, c := range cidrs {
		addr, ok := netip.AddrFromSlice(c.Ip)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s/%d", addr.String(), c.Prefix)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func loadCustomGeoSites(rulesDir string) ([]*routercommon.GeoSite, error) {
	matches, err := filepath.Glob(filepath.Join(rulesDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)

	var sites []*routercommon.GeoSite
	for _, p := range matches {
		domains, _, err := parseClassicalYAML(p)
		if err != nil {
			return nil, err
		}
		domains = dedupDomains(domains)
		code := strings.ToUpper(strings.TrimSuffix(filepath.Base(p), ".yaml"))
		if len(domains) > 0 {
			fmt.Printf("  + custom geosite %-32s %d domains\n", code, len(domains))
			sites = append(sites, &routercommon.GeoSite{
				CountryCode: code,
				Domain:      domains,
			})
		}
	}
	return sites, nil
}

func filterGeoSite(inPath, outPath string, allow map[string]bool, custom []*routercommon.GeoSite, mergeInto map[string][]*routercommon.Domain) (int, int, error) {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return 0, 0, err
	}
	var src routercommon.GeoSiteList
	if err := proto.Unmarshal(raw, &src); err != nil {
		return 0, 0, err
	}

	customCodes := make(map[string]bool)
	for _, c := range custom {
		customCodes[c.CountryCode] = true
	}

	var dst routercommon.GeoSiteList
	found := make(map[string]bool)
	for _, entry := range src.Entry {
		code := strings.ToUpper(entry.CountryCode)
		if allow[code] && !customCodes[code] {
			if extra, ok := mergeInto[code]; ok {
				before := len(entry.Domain)
				entry.Domain = dedupDomains(append(entry.Domain, extra...))
				fmt.Printf("  * merged into %-34s %d -> %d domains\n", code, before, len(entry.Domain))
			}
			dst.Entry = append(dst.Entry, entry)
			found[code] = true
		}
	}

	for k := range allow {
		if !found[k] && !customCodes[k] {
			fmt.Fprintf(os.Stderr, "Warning: geosite %q not found in upstream geosite.dat\n", k)
		}
	}

	dst.Entry = append(dst.Entry, custom...)

	out, err := proto.Marshal(&dst)
	if err != nil {
		return 0, 0, err
	}
	return len(src.Entry), len(dst.Entry), os.WriteFile(outPath, out, 0644)
}

func filterGeoIP(inPath, outPath string, allow map[string]bool, custom []*routercommon.GeoIP) (int, int, error) {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return 0, 0, err
	}
	var src routercommon.GeoIPList
	if err := proto.Unmarshal(raw, &src); err != nil {
		return 0, 0, err
	}

	customCodes := make(map[string]bool)
	for _, c := range custom {
		customCodes[c.CountryCode] = true
	}

	var dst routercommon.GeoIPList
	found := make(map[string]bool)
	for _, entry := range src.Entry {
		code := strings.ToUpper(entry.CountryCode)
		if allow[code] && !customCodes[code] {
			dst.Entry = append(dst.Entry, entry)
			found[code] = true
		}
	}

	for k := range allow {
		if !found[k] && !customCodes[k] {
			fmt.Fprintf(os.Stderr, "Warning: geoip %q not found in upstream geoip.dat\n", k)
		}
	}

	dst.Entry = append(dst.Entry, custom...)

	out, err := proto.Marshal(&dst)
	if err != nil {
		return 0, 0, err
	}
	return len(src.Entry), len(dst.Entry), os.WriteFile(outPath, out, 0644)
}

func main() {
	siteAllow, err := loadList("geosite.txt")
	if err != nil {
		panic(err)
	}
	ipAllow, err := loadList("geoip.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println("Custom geosite categories:")
	customSites, err := loadCustomGeoSites("rules")
	if err != nil {
		panic(err)
	}

	var customIPs []*routercommon.GeoIP
	mergeInto := make(map[string][]*routercommon.Domain)

	// Merge Aethersailor/Custom_OpenClash_Rules if downloaded by CI
	if d, c, err := parseClassicalYAML("upstream_custom_direct.yaml"); err == nil && (len(d) > 0 || len(c) > 0) {
		d = dedupDomains(d)
		fmt.Printf("  + custom geosite %-32s %d domains\n", "CUSTOM-DIRECT", len(d))
		customSites = append(customSites, &routercommon.GeoSite{CountryCode: "CUSTOM-DIRECT", Domain: d})
		if len(c) > 0 {
			c = dedupCIDRs(c)
			fmt.Printf("  + custom geoip   %-32s %d cidrs\n", "CUSTOM-DIRECT", len(c))
			customIPs = append(customIPs, &routercommon.GeoIP{CountryCode: "CUSTOM-DIRECT", Cidr: c})
		}
	}

	if d, _, err := parseClassicalYAML("upstream_custom_proxy.yaml"); err == nil && len(d) > 0 {
		d = dedupDomains(d)
		fmt.Printf("  + custom geosite %-32s %d domains\n", "CUSTOM-PROXY", len(d))
		customSites = append(customSites, &routercommon.GeoSite{CountryCode: "CUSTOM-PROXY", Domain: d})
	}

	var gameDomains []*routercommon.Domain
	var gameCIDRs []*routercommon.CIDR
	for _, fn := range []string{"upstream_steam_cdn.yaml", "upstream_game_cdn.yaml"} {
		if d, c, err := parseClassicalYAML(fn); err == nil {
			gameDomains = append(gameDomains, d...)
			gameCIDRs = append(gameCIDRs, c...)
		}
	}
	if len(gameDomains) > 0 {
		mergeInto["CATEGORY-GAME-PLATFORMS-DOWNLOAD"] = dedupDomains(gameDomains)
	}
	if len(gameCIDRs) > 0 {
		gameCIDRs = dedupCIDRs(gameCIDRs)
		fmt.Printf("  + custom geoip   %-32s %d cidrs\n", "GAME-CDN", len(gameCIDRs))
		customIPs = append(customIPs, &routercommon.GeoIP{CountryCode: "GAME-CDN", Cidr: gameCIDRs})
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		panic(err)
	}

	sTotal, sKept, err := filterGeoSite("geosite.dat", filepath.Join("dist", "geosite.dat"), siteAllow, customSites, mergeInto)
	if err != nil {
		panic(err)
	}
	iTotal, iKept, err := filterGeoIP("geoip.dat", filepath.Join("dist", "geoip.dat"), ipAllow, customIPs)
	if err != nil {
		panic(err)
	}

	fmt.Printf("geosite.dat: kept %d / %d upstream + %d custom categories\n", sKept-len(customSites), sTotal, len(customSites))
	fmt.Printf("geoip.dat:   kept %d / %d upstream + %d custom categories\n", iKept-len(customIPs), iTotal, len(customIPs))
}
