package validate

import (
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strings"

	"gateway-control/internal/model"
)

var (
	namePattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	hostPattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
	allowedMethods = map[string]struct{}{
		"GET": {}, "HEAD": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "OPTIONS": {},
	}
)

type Error struct {
	Details []string
}

func (e *Error) Error() string {
	return strings.Join(e.Details, "; ")
}

func Normalize(in model.ServiceInput) (model.ServiceInput, []string) {
	var errs []string
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Owner = strings.TrimSpace(in.Owner)
	in.Importance = strings.ToLower(strings.TrimSpace(in.Importance))
	if in.Importance == "" {
		in.Importance = model.ImportanceMedium
	}
	if !namePattern.MatchString(in.Name) {
		errs = append(errs, "servis adı küçük harf, rakam ve tire içermeli")
	}
	if in.Owner == "" || len(in.Owner) > 80 {
		errs = append(errs, "sahip zorunlu ve 80 karakteri aşamaz")
	}
	if len(in.Description) > 500 {
		errs = append(errs, "açıklama 500 karakteri aşamaz")
	}
	switch in.Importance {
	case model.ImportanceLow, model.ImportanceMedium, model.ImportanceHigh:
	default:
		errs = append(errs, "önem seviyesi low, medium veya high olmalı")
	}
	if in.EnvironmentID == "" || in.TeamID == "" || in.FleetID == "" {
		errs = append(errs, "ortam, ekip ve fleet zorunlu")
	}
	if len(in.Hostnames) == 0 {
		errs = append(errs, "en az bir hostname gerekli")
	}
	seenHost := map[string]struct{}{}
	for i, host := range in.Hostnames {
		host = strings.ToLower(strings.TrimSpace(host))
		in.Hostnames[i] = host
		if host == "api-gateway.mehmet.tech" {
			errs = append(errs, "api-gateway.mehmet.tech merkezi gateway icin ayrilmistir")
			continue
		}
		if strings.Contains(host, "*") || host == "" {
			errs = append(errs, "joker veya boş hostname kapalı")
			continue
		}
		if !hostPattern.MatchString(host) {
			errs = append(errs, fmt.Sprintf("hostname geçersiz: %s", host))
			continue
		}
		if _, ok := seenHost[host]; ok {
			errs = append(errs, "hostname tekrar ediyor: "+host)
			continue
		}
		seenHost[host] = struct{}{}
	}
	if len(in.Routes) == 0 {
		errs = append(errs, "en az bir rota gerekli")
	}
	seenRoute := map[string]struct{}{}
	for i := range in.Routes {
		route := &in.Routes[i]
		if route.RateLimitRPS<0 || route.RateLimitRPS>10000 {errs=append(errs,"endpoint rate limit 0-10000 istek/saniye olmalı")}
		route.Path = strings.TrimSpace(route.Path)
		route.MatchType = strings.ToLower(strings.TrimSpace(route.MatchType))
		if route.MatchType == "" {
			route.MatchType = model.MatchPrefix
		}
		if route.MatchType != model.MatchPrefix && route.MatchType != model.MatchExact {
			errs = append(errs, "eşleşme prefix veya exact olmalı")
		}
		if !strings.HasPrefix(route.Path, "/") || strings.ContainsAny(route.Path, " \t?#") || strings.Contains(route.Path, "//") {
			errs = append(errs, "yol / ile başlamalı ve boşluk, sorgu veya çift eğik çizgi içeremez")
		}
		methods := make([]string, 0, len(route.Methods))
		for _, method := range route.Methods {
			method = strings.ToUpper(strings.TrimSpace(method))
			if method == "" {
				continue
			}
			if _, ok := allowedMethods[method]; !ok {
				errs = append(errs, "desteklenmeyen method: "+method)
				continue
			}
			methods = append(methods, method)
		}
		route.Methods = methods
		key := route.MatchType + " " + route.Path + " " + strings.Join(methods, ",")
		if _, ok := seenRoute[key]; ok {
			errs = append(errs, "aynı rota tekrar ediyor: "+route.Path)
		}
		seenRoute[key] = struct{}{}

		up := &route.Upstream
		up.Name = strings.TrimSpace(up.Name)
		up.Algorithm = strings.ToLower(strings.TrimSpace(up.Algorithm))
		if up.Name == "" || !namePattern.MatchString(up.Name) {
			errs = append(errs, "upstream adı geçersiz")
		}
		if up.Algorithm == "" {
			up.Algorithm = model.AlgoRoundRobin
		}
		if up.Algorithm != model.AlgoRoundRobin && up.Algorithm != model.AlgoLeastRequest {
			errs = append(errs, "algoritma round_robin veya least_request olmalı")
		}
		if up.ConnectTimeoutMs == 0 {
			up.ConnectTimeoutMs = 2000
		}
		if up.RequestTimeoutMs == 0 {
			up.RequestTimeoutMs = 15000
		}
		if up.ConnectTimeoutMs < 100 || up.ConnectTimeoutMs > 60000 {
			errs = append(errs, "bağlantı zaman aşımı 100 ile 60000 ms arasında olmalı")
		}
		if up.RequestTimeoutMs < 100 || up.RequestTimeoutMs > 600000 {
			errs = append(errs, "istek zaman aşımı 100 ile 600000 ms arasında olmalı")
		}
		if len(up.Endpoints) == 0 {
			errs = append(errs, "upstream en az bir uç nokta içermeli")
		}
		for j := range up.Endpoints {
			ep := &up.Endpoints[j]
			ep.Address = strings.TrimSpace(ep.Address)
			if ep.Weight == 0 {
				ep.Weight = 1
			}
			if ep.Weight < 1 || ep.Weight > 1000 {
				errs = append(errs, "ağırlık 1 ile 1000 arasında olmalı")
			}
			if ep.Port < 1 || ep.Port > 65535 {
				errs = append(errs, "uç nokta portu geçersiz")
			}
			if err := CheckAddress(ep.Address); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	return in, errs
}

func CheckAddress(address string) error {
	if address == "" || strings.Contains(address, "://") || strings.ContainsAny(address, " /\\@") {
		return fmt.Errorf("uç nokta adresi geçersiz")
	}
	host := strings.ToLower(address)
	switch host {
	case "localhost", "metadata.google.internal", "metadata.google.internal.":
		return fmt.Errorf("uç nokta adresi engelli: %s", address)
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if blockedIP(ip) {
			return fmt.Errorf("uç nokta adresi engelli: %s", address)
		}
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && blockedNetIP(ip) {
		return fmt.Errorf("uç nokta adresi engelli: %s", address)
	}
	if strings.HasPrefix(host, "127.") {
		return fmt.Errorf("uç nokta adresi engelli: %s", address)
	}
	if !namePattern.MatchString(host) && !hostPattern.MatchString(host) {
		return fmt.Errorf("uç nokta adresi geçersiz: %s", address)
	}
	return nil
}

func blockedIP(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

func blockedNetIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

type OwnedRoute struct {
	ServiceID string
	Hostname  string
	MatchType string
	Path      string
	Methods   []string
}

func Conflicts(serviceID string, hostnames []string, routes []model.RouteInput, existing []OwnedRoute) []string {
	var errs []string
	ownHosts := map[string]struct{}{}
	for _, host := range hostnames {
		ownHosts[host] = struct{}{}
	}
	for _, item := range existing {
		if item.ServiceID == serviceID {
			continue
		}
		if _, ok := ownHosts[item.Hostname]; ok {
			errs = append(errs, "hostname başka serviste kullanılıyor: "+item.Hostname)
		}
	}
	for _, host := range hostnames {
		for _, route := range routes {
			for _, item := range existing {
				if item.ServiceID == serviceID || item.Hostname != host {
					continue
				}
				if item.MatchType == route.MatchType && item.Path == route.Path && methodsOverlap(item.Methods, route.Methods) {
					errs = append(errs, "rota çakışıyor: "+host+route.Path)
				}
			}
		}
	}
	return unique(errs)
}

func methodsOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, item := range a {
		set[item] = struct{}{}
	}
	for _, item := range b {
		if _, ok := set[item]; ok {
			return true
		}
	}
	return false
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
