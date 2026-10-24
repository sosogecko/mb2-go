package xxtea

import (
	"encoding/base64"
	"encoding/binary"
)

const delta uint32 = 0x9E3779B9

func EncryptString(str, key string) string {
	if str == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString(encrypt([]byte(str), []byte(key)))
}

func encrypt(data, key []byte) []byte {
	v := str2long(data, true)
	k := fixk(str2long(key, false))
	n := len(v) - 1
	z := v[n]
	q := 6 + 52/(n+1)
	var sum uint32
	for q > 0 {
		q--
		sum += delta
		e := (sum >> 2) & 3
		var p int
		for p = 0; p < n; p++ {
			y := v[p+1]
			v[p] = v[p] + mx(sum, y, z, uint32(p), e, k)
			z = v[p]
		}
		y := v[0]
		v[n] = v[n] + mx(sum, y, z, uint32(p), e, k)
		z = v[n]
	}
	return long2str(v, false)
}

func mx(sum, y, z, p, e uint32, k []uint32) uint32 {
	return (((z >> 5) & 0x07ffffff) ^ y<<2) + (((y >> 3) & 0x1fffffff) ^ z<<4) ^ ((sum ^ y) + (k[(p&3)^e] ^ z))
}

func fixk(k []uint32) []uint32 {
	if len(k) < 4 {
		nk := make([]uint32, 4)
		copy(nk, k)
		return nk
	}
	return k
}

func str2long(s []byte, w bool) []uint32 {
	n := len(s)
	pad := (4 - n%4) & 3
	buf := make([]byte, n+pad)
	copy(buf, s)
	count := len(buf) / 4
	v := make([]uint32, count)
	for i := 0; i < count; i++ {
		v[i] = binary.LittleEndian.Uint32(buf[i*4:])
	}
	if w {
		v = append(v, uint32(n))
	}
	return v
}

func long2str(v []uint32, w bool) []byte {
	n := len(v)
	s := make([]byte, n*4)
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(s[i*4:], v[i])
	}
	if w {
		m := int(v[n-1])
		if m < 0 || m > n*4 {
			return []byte{}
		}
		return s[:m]
	}
	return s
}
