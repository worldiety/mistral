// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import "testing"

func TestRange_Interval(t *testing.T) {
	tests := []struct {
		name    string
		r       Range
		wantMin int64
		wantMax int64
		wantErr bool
	}{
		{
			"utc",
			"[2020-11-13 12:55:52,2020-11-13 12:56:26]@Etc/UTC",
			1605272152,
			1605272186,
			false,
		},

		{
			"utc+whitespace",
			"  [  2020-11-13 12:55:52  ,   2020-11-13 12:56:26   ]  @  Etc/UTC  ",
			1605272152,
			1605272186,
			false,
		},

		{
			"germany-inc-both",
			"[2020-11-13 14:15:00,2020-11-13 14:20:00]@Europe/Berlin",
			1605273300,
			1605273600,
			false,
		},

		{
			"germany-exl-left",
			"(2020-11-13 14:15:00,2020-11-13 14:20:00]@Europe/Berlin",
			1605273301,
			1605273600,
			false,
		},

		{
			"germany-exl-right",
			"[2020-11-13 14:15:00,2020-11-13 14:20:00)@Europe/Berlin",
			1605273300,
			1605273599,
			false,
		},

		{
			"germany-exl-both",
			"(2020-11-13 14:15:00,2020-11-13 14:20:00)@Europe/Berlin",
			1605273301,
			1605273599,
			false,
		},

		{
			"err-0",
			"([2020-11-13 14:15:00,2020-11-13 14:20:00)@Europe/Berlin",
			-1,
			-1,
			true,
		},

		{
			"err-1",
			"(2020-11-13 14:15:00,2020-11-13 14:20:00)@",
			-1,
			-1,
			true,
		},

		{
			"err-2",
			"(2020-11-13 14:15:00,,2020-11-13 14:20:00)@",
			-1,
			-1,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax, err := tt.r.Interval()
			if (err != nil) != tt.wantErr {
				t.Errorf("Interval() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if gotMin != tt.wantMin {
				t.Errorf("Interval() gotMin = %v, want %v", gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("Interval() gotMax = %v, want %v", gotMax, tt.wantMax)
			}
		})
	}
}
