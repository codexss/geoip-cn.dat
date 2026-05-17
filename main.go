package main

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func pbVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbLD(b []byte, field int, data []byte) []byte {
	b = pbVarint(b, uint64(field<<3|2))
	b = pbVarint(b, uint64(len(data)))
	return append(b, data...)
}

func pbVF(b []byte, field int, v uint64) []byte {
	b = pbVarint(b, uint64(field<<3|0))
	return pbVarint(b, v)
}

func main() {
	fh, err := os.Open("ipinfo_lite.csv")
	if err != nil {
		log.Fatalf("打开文件失败: %v", err)
	}
	defer fh.Close()

	reader := csv.NewReader(bufio.NewReader(fh))
	reader.Read() // 跳过表头

	var cidrs []byte
	count := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || !strings.EqualFold(strings.TrimSpace(record[2]), "CN") {
			continue
		}

		networkStr := record[0]
		if !strings.Contains(networkStr, "/") {
			if ip := net.ParseIP(networkStr); ip != nil {
				if ip.To4() != nil {
					networkStr += "/32"
				} else {
					networkStr += "/128"
				}
			}
		}

		_, ipNet, err := net.ParseCIDR(networkStr)
		if err != nil {
			continue
		}

		ip := ipNet.IP
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}
		ones, _ := ipNet.Mask.Size()

		var cidr []byte
		cidr = pbLD(cidr, 1, ip)
		cidr = pbVF(cidr, 2, uint64(ones))
		cidrs = pbLD(cidrs, 2, cidr)
		count++
	}

	// GeoIP{ country_code="CN", cidr=... }
	var geoip []byte
	geoip = pbLD(geoip, 1, []byte("CN"))
	geoip = append(geoip, cidrs...)

	// GeoIPList{ entry=geoip }
	out := pbLD(nil, 1, geoip)

	if err := os.WriteFile("geoip-cn.dat", out, 0o644); err != nil {
		log.Fatalf("写入失败: %v", err)
	}
	log.Printf("geoip-cn.dat 生成成功，共 %d 条 CN 记录", count)
}