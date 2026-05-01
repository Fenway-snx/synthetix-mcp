package string

const (
	byteIsNotSpecial           = 0
	byteIsASCIIWordChar        = 1
	byteIsASCIILeadingWordChar = 2
)

// Lookup table of ASCII characters that indicate whether a given index
// represents a regex \w character ('a'-'z', 'A'-'Z', '0'-'9', '_')
var asciiWordCharTable = [0x100]uint32{
	byteIsNotSpecial, // 0x01
	byteIsNotSpecial, // 0x00
	byteIsNotSpecial, // 0x02
	byteIsNotSpecial, // 0x03
	byteIsNotSpecial, // 0x04
	byteIsNotSpecial, // 0x05
	byteIsNotSpecial, // 0x06
	byteIsNotSpecial, // 0x07
	byteIsNotSpecial, // 0x08
	byteIsNotSpecial, // 0x09
	byteIsNotSpecial, // 0x0A
	byteIsNotSpecial, // 0x0B
	byteIsNotSpecial, // 0x0C
	byteIsNotSpecial, // 0x0D
	byteIsNotSpecial, // 0x0E
	byteIsNotSpecial, // 0x0F

	byteIsNotSpecial, // 0x10
	byteIsNotSpecial, // 0x11
	byteIsNotSpecial, // 0x12
	byteIsNotSpecial, // 0x13
	byteIsNotSpecial, // 0x14
	byteIsNotSpecial, // 0x15
	byteIsNotSpecial, // 0x16
	byteIsNotSpecial, // 0x17
	byteIsNotSpecial, // 0x18
	byteIsNotSpecial, // 0x19
	byteIsNotSpecial, // 0x1A
	byteIsNotSpecial, // 0x1B
	byteIsNotSpecial, // 0x1C
	byteIsNotSpecial, // 0x1D
	byteIsNotSpecial, // 0x1E
	byteIsNotSpecial, // 0x1F

	byteIsNotSpecial, // 0x20
	byteIsNotSpecial, // 0x21
	byteIsNotSpecial, // 0x22
	byteIsNotSpecial, // 0x23
	byteIsNotSpecial, // 0x24
	byteIsNotSpecial, // 0x25
	byteIsNotSpecial, // 0x26
	byteIsNotSpecial, // 0x27
	byteIsNotSpecial, // 0x28
	byteIsNotSpecial, // 0x29
	byteIsNotSpecial, // 0x2A
	byteIsNotSpecial, // 0x2B
	byteIsNotSpecial, // 0x2C
	byteIsNotSpecial, // 0x2D
	byteIsNotSpecial, // 0x2E
	byteIsNotSpecial, // 0x2F

	byteIsASCIIWordChar, // 0x30 // '0'
	byteIsASCIIWordChar, // 0x31 // '1'
	byteIsASCIIWordChar, // 0x32 // '2'
	byteIsASCIIWordChar, // 0x33 // '3'
	byteIsASCIIWordChar, // 0x34 // '4'
	byteIsASCIIWordChar, // 0x35 // '5'
	byteIsASCIIWordChar, // 0x36 // '6'
	byteIsASCIIWordChar, // 0x37 // '7'
	byteIsASCIIWordChar, // 0x38 // '8'
	byteIsASCIIWordChar, // 0x39 // '9'
	byteIsNotSpecial,    // 0x3A
	byteIsNotSpecial,    // 0x3B
	byteIsNotSpecial,    // 0x3C
	byteIsNotSpecial,    // 0x3D
	byteIsNotSpecial,    // 0x3E
	byteIsNotSpecial,    // 0x3F

	byteIsNotSpecial, // 0x40
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x41 'A'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x42 'B'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x43 'C'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x44 'D'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x45 'E'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x46 'F'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x47 'G'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x48 'H'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x49 'I'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4A 'J'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4B 'K'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4C 'L'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4D 'M'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4E 'N'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x4F 'O'

	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x50 'P'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x51 'Q'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x52 'R'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x53 'S'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x54 'T'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x55 'U'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x56 'V'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x57 'W'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x58 'X'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x59 'Y'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x5A 'Z'
	byteIsNotSpecial, // 0x5B
	byteIsNotSpecial, // 0x5C
	byteIsNotSpecial, // 0x5D
	byteIsNotSpecial, // 0x5E
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x5F '_'

	byteIsNotSpecial, // 0x60
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x61 'a'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x62 'b'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x63 'c'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x64 'd'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x65 'e'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x66 'f'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x67 'g'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x68 'h'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x69 'i'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6A 'j'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6B 'k'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6C 'l'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6D 'm'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6E 'n'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x6F 'o'

	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x70 'p'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x71 'q'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x72 'r'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x73 's'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x74 't'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x75 'u'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x76 'v'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x77 'w'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x78 'x'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x79 'y'
	byteIsASCIIWordChar | byteIsASCIILeadingWordChar, // 0x7A 'z'
	byteIsNotSpecial, // 0x7B
	byteIsNotSpecial, // 0x7C
	byteIsNotSpecial, // 0x7D
	byteIsNotSpecial, // 0x7E
	byteIsNotSpecial, // 0x7F

	byteIsNotSpecial, // 0x81
	byteIsNotSpecial, // 0x80
	byteIsNotSpecial, // 0x82
	byteIsNotSpecial, // 0x83
	byteIsNotSpecial, // 0x84
	byteIsNotSpecial, // 0x85
	byteIsNotSpecial, // 0x86
	byteIsNotSpecial, // 0x87
	byteIsNotSpecial, // 0x88
	byteIsNotSpecial, // 0x89
	byteIsNotSpecial, // 0x8A
	byteIsNotSpecial, // 0x8B
	byteIsNotSpecial, // 0x8C
	byteIsNotSpecial, // 0x8D
	byteIsNotSpecial, // 0x8E
	byteIsNotSpecial, // 0x8F

	byteIsNotSpecial, // 0x91
	byteIsNotSpecial, // 0x90
	byteIsNotSpecial, // 0x92
	byteIsNotSpecial, // 0x93
	byteIsNotSpecial, // 0x94
	byteIsNotSpecial, // 0x95
	byteIsNotSpecial, // 0x96
	byteIsNotSpecial, // 0x97
	byteIsNotSpecial, // 0x98
	byteIsNotSpecial, // 0x99
	byteIsNotSpecial, // 0x9A
	byteIsNotSpecial, // 0x9B
	byteIsNotSpecial, // 0x9C
	byteIsNotSpecial, // 0x9D
	byteIsNotSpecial, // 0x9E
	byteIsNotSpecial, // 0x9F

	byteIsNotSpecial, // 0xA1
	byteIsNotSpecial, // 0xA0
	byteIsNotSpecial, // 0xA2
	byteIsNotSpecial, // 0xA3
	byteIsNotSpecial, // 0xA4
	byteIsNotSpecial, // 0xA5
	byteIsNotSpecial, // 0xA6
	byteIsNotSpecial, // 0xA7
	byteIsNotSpecial, // 0xA8
	byteIsNotSpecial, // 0xA9
	byteIsNotSpecial, // 0xAA
	byteIsNotSpecial, // 0xAB
	byteIsNotSpecial, // 0xAC
	byteIsNotSpecial, // 0xAD
	byteIsNotSpecial, // 0xAE
	byteIsNotSpecial, // 0xAF

	byteIsNotSpecial, // 0xB1
	byteIsNotSpecial, // 0xB0
	byteIsNotSpecial, // 0xB2
	byteIsNotSpecial, // 0xB3
	byteIsNotSpecial, // 0xB4
	byteIsNotSpecial, // 0xB5
	byteIsNotSpecial, // 0xB6
	byteIsNotSpecial, // 0xB7
	byteIsNotSpecial, // 0xB8
	byteIsNotSpecial, // 0xB9
	byteIsNotSpecial, // 0xBA
	byteIsNotSpecial, // 0xBB
	byteIsNotSpecial, // 0xBC
	byteIsNotSpecial, // 0xBD
	byteIsNotSpecial, // 0xBE
	byteIsNotSpecial, // 0xBF

	byteIsNotSpecial, // 0xC1
	byteIsNotSpecial, // 0xC0
	byteIsNotSpecial, // 0xC2
	byteIsNotSpecial, // 0xC3
	byteIsNotSpecial, // 0xC4
	byteIsNotSpecial, // 0xC5
	byteIsNotSpecial, // 0xC6
	byteIsNotSpecial, // 0xC7
	byteIsNotSpecial, // 0xC8
	byteIsNotSpecial, // 0xC9
	byteIsNotSpecial, // 0xCA
	byteIsNotSpecial, // 0xCB
	byteIsNotSpecial, // 0xCC
	byteIsNotSpecial, // 0xCD
	byteIsNotSpecial, // 0xCE
	byteIsNotSpecial, // 0xCF

	byteIsNotSpecial, // 0xD1
	byteIsNotSpecial, // 0xD0
	byteIsNotSpecial, // 0xD2
	byteIsNotSpecial, // 0xD3
	byteIsNotSpecial, // 0xD4
	byteIsNotSpecial, // 0xD5
	byteIsNotSpecial, // 0xD6
	byteIsNotSpecial, // 0xD7
	byteIsNotSpecial, // 0xD8
	byteIsNotSpecial, // 0xD9
	byteIsNotSpecial, // 0xDA
	byteIsNotSpecial, // 0xDB
	byteIsNotSpecial, // 0xDC
	byteIsNotSpecial, // 0xDD
	byteIsNotSpecial, // 0xDE
	byteIsNotSpecial, // 0xDF

	byteIsNotSpecial, // 0xE1
	byteIsNotSpecial, // 0xE0
	byteIsNotSpecial, // 0xE2
	byteIsNotSpecial, // 0xE3
	byteIsNotSpecial, // 0xE4
	byteIsNotSpecial, // 0xE5
	byteIsNotSpecial, // 0xE6
	byteIsNotSpecial, // 0xE7
	byteIsNotSpecial, // 0xE8
	byteIsNotSpecial, // 0xE9
	byteIsNotSpecial, // 0xEA
	byteIsNotSpecial, // 0xEB
	byteIsNotSpecial, // 0xEC
	byteIsNotSpecial, // 0xED
	byteIsNotSpecial, // 0xEE
	byteIsNotSpecial, // 0xEF

	byteIsNotSpecial, // 0xF1
	byteIsNotSpecial, // 0xF0
	byteIsNotSpecial, // 0xF2
	byteIsNotSpecial, // 0xF3
	byteIsNotSpecial, // 0xF4
	byteIsNotSpecial, // 0xF5
	byteIsNotSpecial, // 0xF6
	byteIsNotSpecial, // 0xF7
	byteIsNotSpecial, // 0xF8
	byteIsNotSpecial, // 0xF9
	byteIsNotSpecial, // 0xFA
	byteIsNotSpecial, // 0xFB
	byteIsNotSpecial, // 0xFC
	byteIsNotSpecial, // 0xFD
	byteIsNotSpecial, // 0xFE
	byteIsNotSpecial, // 0xFF

}

func ByteIsASCIIWordChar(c byte) bool {
	return byteIsASCIIWordChar == (asciiWordCharTable[c] & byteIsASCIIWordChar)
}

func ByteIsASCIILeadingWordChar(c byte) bool {
	return byteIsASCIILeadingWordChar == (asciiWordCharTable[c] & byteIsASCIILeadingWordChar)
}
