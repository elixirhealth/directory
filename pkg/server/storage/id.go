package storage

import (
	"encoding/base32"
	"io"

	"github.com/pkg/errors"
)

var (
	// ErrPrefixTooLong indicates when the ID prefix is more than one character
	ErrPrefixTooLong = errors.New("ID prefix longer than 1 character")

	// ErrInvalidPrefix indicates when the prefix character is not base 32.
	ErrInvalidPrefix = errors.New("prefix character not in base 32")

	// ErrIncorrectChecksum indicates when the checksum is incorrect for the rest of the ID>
	ErrIncorrectChecksum = errors.New("last character is incorrect checksum")

	// base32Encoder and base32Decoder come from base32.StdEncoding but are defined here as
	// well for purpose of calculating Luhn checksum
	base32Encoder = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	base32Decoder = getDecoder(base32Encoder)
)

const (
	invalidCodePoint = 0xFF
)

type ChecksumIDChecker interface {
	// Check confirms that the right-most character properly checksums the rest of the id,
	// returning a non-nil error if it does not.
	Check(id string) error
}

// ChecksumIDGenerator creates base-32 IDs with a Luhn checksum character at the end.
type ChecksumIDGenerator interface {
	ChecksumIDChecker

	// Generate returns an ID with the given (usually 1-character) prefix.
	Generate(prefix string) (string, error)
}

type naiveIDGenerator struct {
	entropy io.Reader
	length  int
}

// NewNaiveIDGenerator returns an ChecksumIDGenerator that produces IDs of the given length from
// the given source of entropy. It is naive in that it does not check whether the generated ID
// exists or not.
func NewNaiveIDGenerator(entropy io.Reader, length int) ChecksumIDGenerator {
	return &naiveIDGenerator{
		entropy: entropy,
		length:  length,
	}
}

func (g *naiveIDGenerator) Generate(prefix string) (string, error) {
	if len(prefix) > 1 {
		return "", ErrPrefixTooLong
	}
	if len(prefix) == 1 && base32Decoder[prefix[0]] == invalidCodePoint {
		return "", ErrInvalidPrefix
	}
	randB32, err := readBase32(g.entropy, g.length-len(prefix)-1)
	if err != nil {
		return "", err
	}
	raw := prefix + randB32
	checksum := calcChecksum(raw, base32Encoder, base32Decoder)
	return raw + checksum, nil
}

func (g *naiveIDGenerator) Check(id string) error {
	if ok := checkChecksum(id, base32Encoder, base32Decoder); !ok {
		return ErrIncorrectChecksum
	}
	return nil
}

func getDecoder(encoder string) [256]byte {
	var decoder [256]byte
	for i := 0; i < len(decoder); i++ {
		decoder[i] = invalidCodePoint
	}
	for i := 0; i < len(encoder); i++ {
		decoder[encoder[i]] = byte(i)
	}
	return decoder
}

func readBase32(entropy io.Reader, n int) (string, error) {
	nBytes := base32.StdEncoding.DecodedLen(n + 8)
	decoded := make([]byte, nBytes)
	if _, err := entropy.Read(decoded); err != nil {
		return "", err
	}
	encoded := base32.NewEncoding(base32Encoder).EncodeToString(decoded)[:n]
	return encoded, nil
}

// calcChecksum calculates the (length 1) Luhn check character for the given raw string. It uses the
// base32Encoder string that represents the consecutive code points and the inverse mapping from
// rune to code point.
// reference: https://en.wikipedia.org/wiki/Luhn_mod_N_algorithm
func calcChecksum(raw string, encoder string, decodeMap [256]byte) string {
	factor := 2 // start with 2 since right-most character is checksum
	sum := 0
	base := len(encoder)
	for i := len(raw) - 1; i >= 0; i-- {
		codePoint := int(decodeMap[raw[i]])
		addend := factor * codePoint

		// sum the digits of the "addend" as expressed in base
		addend = addend/base + (addend % base)
		sum += addend

		if factor == 2 {
			factor = 1
		} else {
			factor = 2
		}
	}
	remainder := sum % base
	checksum := (base - remainder) % base
	return string(encoder[checksum])
}

// checkChecksum checks that the given value's checksum equals the expected value
func checkChecksum(value string, encoder string, decodeMap [256]byte) bool {
	raw := value[:len(value)-1]
	checksum := value[len(value)-1]
	return calcChecksum(raw, encoder, decodeMap) == string(checksum)
}
