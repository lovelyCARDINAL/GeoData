package main

import (
	"bufio"
	"fmt"
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

func loadCustomGeoSites(rulesDir string) ([]*routercommon.GeoSite, error) {
	matches, err := filepath.Glob(filepath.Join(rulesDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)

	var sites []*routercommon.GeoSite
	for _, p := range matches {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		code := strings.ToUpper(strings.TrimSuffix(filepath.Base(p), ".yaml"))
		site := &routercommon.GeoSite{CountryCode: code}

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "- ") {
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			item = strings.Trim(item, "\"'")
			parts := strings.SplitN(item, ",", 2)
			if len(parts) != 2 {
				continue
			}
			kind := strings.ToUpper(strings.TrimSpace(parts[0]))
			val := strings.TrimSpace(parts[1])
			switch kind {
			case "DOMAIN":
				site.Domain = append(site.Domain, &routercommon.Domain{
					Type:  routercommon.Domain_Full,
					Value: val,
				})
			case "DOMAIN-SUFFIX":
				site.Domain = append(site.Domain, &routercommon.Domain{
					Type:  routercommon.Domain_RootDomain,
					Value: val,
				})
			case "DOMAIN-KEYWORD":
				site.Domain = append(site.Domain, &routercommon.Domain{
					Type:  routercommon.Domain_Plain,
					Value: val,
				})
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return nil, err
		}
		if len(site.Domain) > 0 {
			fmt.Printf("  + custom geosite %-16s %d domains\n", code, len(site.Domain))
			sites = append(sites, site)
		}
	}
	return sites, nil
}

func filterGeoSite(inPath, outPath string, allow map[string]bool, custom []*routercommon.GeoSite) (int, int, error) {
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

func filterGeoIP(inPath, outPath string, allow map[string]bool) (int, int, error) {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return 0, 0, err
	}
	var src routercommon.GeoIPList
	if err := proto.Unmarshal(raw, &src); err != nil {
		return 0, 0, err
	}

	var dst routercommon.GeoIPList
	found := make(map[string]bool)
	for _, entry := range src.Entry {
		code := strings.ToUpper(entry.CountryCode)
		if allow[code] {
			dst.Entry = append(dst.Entry, entry)
			found[code] = true
		}
	}

	for k := range allow {
		if !found[k] {
			fmt.Fprintf(os.Stderr, "Warning: geoip %q not found in upstream geoip.dat\n", k)
		}
	}

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

	if err := os.MkdirAll("dist", 0755); err != nil {
		panic(err)
	}

	sTotal, sKept, err := filterGeoSite("geosite.dat", filepath.Join("dist", "geosite.dat"), siteAllow, customSites)
	if err != nil {
		panic(err)
	}
	iTotal, iKept, err := filterGeoIP("geoip.dat", filepath.Join("dist", "geoip.dat"), ipAllow)
	if err != nil {
		panic(err)
	}

	fmt.Printf("geosite.dat: kept %d / %d upstream + %d custom categories\n", sKept-len(customSites), sTotal, len(customSites))
	fmt.Printf("geoip.dat:   kept %d / %d categories\n", iKept, iTotal)
}
