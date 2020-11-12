package net

import "strings"

func AnalyIP(ip string) (network, ip_address string) {
	i := strings.Split(ip, "/")
	ip_address_with_mask := i[len(i)-1]
	network = strings.Replace(ip_address_with_mask, "-", "/", -1)
	ip_address = strings.Split(ip_address_with_mask, "-")[0]
	return
}
