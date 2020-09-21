package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rancher/octopus/adaptors/ble/api/v1alpha1"
)

func TestBitsOperations(t *testing.T) {
	type input struct {
		converter *v1alpha1.BluetoothDataConverter
		data      []byte
	}
	type output struct {
		result uint64
		err    error
	}
	var testCases = []struct {
		given    input
		expected output
	}{
		{
			given: input{
				data: []byte{0x03, 0x00, 0x01},
				converter: &v1alpha1.BluetoothDataConverter{
					StartIndex:        1,
					EndIndex:          2,
					ShiftLeft:         1,
					ShiftRight:        0,
					OrderOfOperations: nil,
				},
			},
			expected: output{
				result: 2,
				err:    nil,
			},
		},
		{
			given: input{
				data: []byte{0x00, 0x01, 0x02, 0x03},
				converter: &v1alpha1.BluetoothDataConverter{
					StartIndex:        0,
					EndIndex:          3,
					ShiftLeft:         0,
					ShiftRight:        2,
					OrderOfOperations: nil,
				},
			},
			expected: output{
				result: 72,
				err:    nil,
			},
		},
	}
	for i, tc := range testCases {
		var actual output
		actual.result, actual.err = doBitsOperations(tc.given.data, tc.given.converter)
		assert.Equal(t, tc.expected, actual, "case %d", i)
	}
}
