package adapter

import "testing"

func TestParseUptimeToSeconds(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect int64
	}{
		{name: "clock only", input: "00:05:30", expect: 330},
		{name: "days plus clock", input: "45d00:00:00", expect: 3888000},
		{name: "weeks days clock", input: "1w2d03:04:05", expect: 788645},
		{name: "routeros unit suffixes", input: "4w6d22h24m59s", expect: 3018299},
		{name: "minutes seconds suffix", input: "12m30s", expect: 750},
		{name: "zero seconds", input: "0s", expect: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseUptimeToSeconds(test.input)
			if got != test.expect {
				t.Fatalf("parseUptimeToSeconds(%q) = %d, want %d", test.input, got, test.expect)
			}
		})
	}
}
