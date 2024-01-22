package aztec

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takt-corp/barcode/utils"
)

func bitStr(bl *utils.BitList) string {
	buf := new(bytes.Buffer)

	for i := 0; i < bl.Len(); i++ {
		if bl.GetBit(i) {
			buf.WriteRune('X')
		} else {
			buf.WriteRune('.')
		}
	}
	return buf.String()
}

func testHighLevelEncodeString(t *testing.T, s, expectedBits string) {
	bits := highlevelEncode([]byte(s))
	result := bitStr(bits)
	expectedBits = strings.Replace(expectedBits, " ", "", -1)

	if result != expectedBits {
		t.Errorf("invalid result for highlevelEncode(%q). Got:\n%s", s, result)
	}
}
func testHighLevelEncodeStringCnt(t *testing.T, s string, expectedBitCnt int) {
	bits := highlevelEncode([]byte(s))

	if bits.Len() != expectedBitCnt {
		t.Errorf("invalid result for highlevelEncode(%q). Got %d, expected %d bits", s, bits.Len(), expectedBitCnt)
	}
}

func Test_HighLevelEncode(t *testing.T) {
	testHighLevelEncodeString(t, "A. b.",
		// 'A'  P/S   '. ' L/L    b    D/L    '.'
		"...X. ..... ...XX XXX.. ...XX XXXX. XX.X")
	testHighLevelEncodeString(t, "Lorem ipsum.",
		// 'L'  L/L   'o'   'r'   'e'   'm'   ' '   'i'   'p'   's'   'u'   'm'   D/L   '.'
		".XX.X XXX.. X.... X..XX ..XX. .XXX. ....X .X.X. X...X X.X.. X.XX. .XXX. XXXX. XX.X")
	testHighLevelEncodeString(t, "Lo. Test 123.",
		// 'L'  L/L   'o'   P/S   '. '  U/S   'T'   'e'   's'   't'    D/L   ' '  '1'  '2'  '3'  '.'
		".XX.X XXX.. X.... ..... ...XX XXX.. X.X.X ..XX. X.X.. X.X.X  XXXX. ...X ..XX .X.. .X.X XX.X")
	testHighLevelEncodeString(t, "Lo...x",
		// 'L'  L/L   'o'   D/L   '.'  '.'  '.'  U/L  L/L   'x'
		".XX.X XXX.. X.... XXXX. XX.X XX.X XX.X XXX. XXX.. XX..X")
	testHighLevelEncodeString(t, ". x://abc/.",
		//P/S   '. '  L/L   'x'   P/S   ':'   P/S   '/'   P/S   '/'   'a'   'b'   'c'   P/S   '/'   D/L   '.'
		"..... ...XX XXX.. XX..X ..... X.X.X ..... X.X.. ..... X.X.. ...X. ...XX ..X.. ..... X.X.. XXXX. XX.X")
	// Uses Binary/Shift rather than Lower/Shift to save two bits.
	testHighLevelEncodeString(t, "ABCdEFG",
		//'A'   'B'   'C'   B/S    =1    'd'     'E'   'F'   'G'
		"...X. ...XX ..X.. XXXXX ....X .XX..X.. ..XX. ..XXX .X...")

	testHighLevelEncodeStringCnt(t,
		// Found on an airline boarding pass.  Several stretches of Binary shift are
		// necessary to keep the bitcount so low.
		"09  UAG    ^160MEUCIQC0sYS/HpKxnBELR1uB85R20OoqqwFGa0q2uEi"+
			"Ygh6utAIgLl1aBVM4EOTQtMQQYH9M2Z3Dp4qnA/fwWuQ+M8L3V8U=",
		823)
}

func Test_HighLevelEncodeBinary(t *testing.T) {
	// binary short form single byte
	testHighLevelEncodeString(t, "N\u0000N",
		// 'N'  B/S    =1   '\0'      N
		".XXXX XXXXX ....X ........ .XXXX") // Encode "N" in UPPER

	testHighLevelEncodeString(t, "N\u0000n",
		// 'N'  B/S    =2   '\0'       'n'
		".XXXX XXXXX ...X. ........ .XX.XXX.") // Encode "n" in BINARY

	// binary short form consecutive bytes
	testHighLevelEncodeString(t, "N\x00\x80 A",
		// 'N'  B/S    =2    '\0'    \u0080   ' '  'A'
		".XXXX XXXXX ...X. ........ X....... ....X ...X.")

	// binary skipping over single character
	testHighLevelEncodeString(t, "\x00a\xFF\x80 A",
		// B/S  =4    '\0'      'a'     '\3ff'   '\200'   ' '   'A'
		"XXXXX ..X.. ........ .XX....X XXXXXXXX X....... ....X ...X.")

	// getting into binary mode from digit mode
	testHighLevelEncodeString(t, "1234\u0000",
		//D/L   '1'  '2'  '3'  '4'  U/L  B/S    =1    \0
		"XXXX. ..XX .X.. .X.X .XX. XXX. XXXXX ....X ........")

	// Create a string in which every character requires binary
	sb := new(bytes.Buffer)
	for i := 0; i <= 3000; i++ {
		sb.WriteByte(byte(128 + (i % 30)))
	}

	// Test the output generated by Binary/Switch, particularly near the
	// places where the encoding changes: 31, 62, and 2047+31=2078
	for _, i := range []int{1, 2, 3, 10, 29, 30, 31, 32, 33, 60, 61, 62, 63, 64, 2076, 2077, 2078, 2079, 2080, 2100} {
		// This is the expected length of a binary string of length "i"
		expectedLength := (8 * i)
		switch {
		case i <= 31:
			expectedLength += 10
		case i <= 62:
			expectedLength += 20
		case i <= 2078:
			expectedLength += 21
		default:
			expectedLength += 31
		}
		data := string(sb.Bytes()[:i])

		// Verify that we are correct about the length.
		testHighLevelEncodeStringCnt(t, data, expectedLength)
		if i != 1 && i != 32 && i != 2079 {
			// The addition of an 'a' at the beginning or end gets merged into the binary code
			// in those cases where adding another binary character only adds 8 or 9 bits to the result.
			// So we exclude the border cases i=1,32,2079
			// A lower case letter at the beginning will be merged into binary mode
			testHighLevelEncodeStringCnt(t, "a"+string(sb.Bytes()[:i-1]), expectedLength)
			// A lower case letter at the end will also be merged into binary mode
			testHighLevelEncodeStringCnt(t, string(sb.Bytes()[:i-1])+"a", expectedLength)
		}
		// A lower case letter at both ends will enough to latch us into LOWER.
		testHighLevelEncodeStringCnt(t, "a"+data+"b", expectedLength+15)
	}
}
