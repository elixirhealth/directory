package storage

import (
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	idLength = 9
)

func TestNaiveIDGenerator_Generate_ok(t *testing.T) {
	prefix := "P"
	rng := rand.New(rand.NewSource(0))
	g := NewNaiveIDGenerator(rng, idLength)
	id, err := g.Generate(prefix)
	assert.Nil(t, err)
	assert.Equal(t, idLength, len(id))
	assert.Equal(t, prefix, string(id[0]))
	assert.True(t, checkChecksum(id, base32Encoder, base32Decoder))
}

func TestNaiveIDGenerator_Generate_err(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	prefix := "P"
	g := NewNaiveIDGenerator(rng, idLength)

	id, err := g.Generate("pp")
	assert.Equal(t, ErrPrefixTooLong, err)
	assert.Empty(t, id)

	id, err = g.Generate("p")
	assert.Equal(t, ErrInvalidPrefix, err)
	assert.Empty(t, id)

	g = &naiveIDGenerator{
		entropy: &fixedReader{err: errors.New("some Read error")},
		length:  idLength,
	}
	id, err = g.Generate(prefix)
	assert.NotNil(t, err)
	assert.Empty(t, id)
}

func TestNaiveIDGenerator_Check(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	g := NewNaiveIDGenerator(rng, idLength)

	err := g.Check("PAGKP3QXS")
	assert.Nil(t, err)

	err = g.Check("PAGKP3QXR")
	assert.Equal(t, ErrIncorrectChecksum, err)
}

func TestCalcChecksum(t *testing.T) {
	// case from https://en.wikipedia.org/wiki/Luhn_mod_N_algorithm
	encoder := "abcdef"
	checksum := calcChecksum("abcdef", encoder, getDecoder(encoder))
	assert.Equal(t, "e", checksum)
}

type fixedReader struct {
	err error
}

func (f *fixedReader) Read(p []byte) (n int, err error) {
	return 0, f.err
}
