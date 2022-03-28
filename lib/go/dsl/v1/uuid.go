// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// UUID represents 16 byte for a UUID.
type UUID [16]byte

// String returns the typical UUID representation.
func (u UUID) String() string {
	tmp := make([]byte, 36)
	hex.Encode(tmp, u[:4])
	tmp[8] = '-'
	hex.Encode(tmp[9:13], u[4:6])
	tmp[13] = '-'
	hex.Encode(tmp[14:18], u[6:8])
	tmp[18] = '-'
	hex.Encode(tmp[19:23], u[8:10])
	tmp[23] = '-'
	hex.Encode(tmp[24:], u[10:])
	return string(tmp)
}

// MarshalText renders the UUID properly into JSON.
func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UUID) UnmarshalText(data []byte) error {
	uuid, err := ParseUUID(string(data))
	if err != nil {
		return err
	}

	*u = uuid
	return nil
}

// ParseUUID can only parse UUIDs like 12da0b4c-8f1e-4897-842f-3487849dfba6.
// However, intentionally any hex combination can be parsed, even if that does not
// represent a real UUID. This allows the intended misuse of the full 16 byte with arbitrary content.
func ParseUUID(text string) (UUID, error) {
	var u UUID
	text = strings.ReplaceAll(text, "-", "")
	buf, err := hex.DecodeString(text)
	if err != nil {
		return u, fmt.Errorf("no valid hex encoding: %w", err)
	}

	copy(u[:], buf)
	return u, nil
}

// NewUUID creates a new secure type 4 UUID or panics.
func NewUUID() UUID {
	var uuid UUID
	_, err := rand.Read(uuid[:])
	if err != nil {
		panic(err)
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return uuid
}
