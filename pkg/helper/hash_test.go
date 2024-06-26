package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Sha512sum_happy(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "password",
			expected: "b109f3bbbc244eb82441917ed06d618b9008dd09b3befd1b5e07394c706a8bb980b1d7785e5976ec049b46df5f1326af5a2ea6d103fd07c95385ffab0cacbc86",
		},
		{
			input:    "123456",
			expected: "ba3253876aed6bc22d4a6ff53d8406c6ad864195ed144ab5c87621b6c233b548baeae6956df346ec8c17f5ea10f35ee3cbc514797ed7ddd3145464e2a0bab413",
		},
		{
			input:    "hello world",
			expected: "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Sha512sum(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
