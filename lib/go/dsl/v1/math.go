// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import "time"

// Math provides a polymorphic entry point (vtable) for a bunch of intrinsically optimized math implementations.
var Math Intrinsics = mathStub{}

// Intrinsics defines the vtable for all required math primitives used by the MiEL v1 api.
type Intrinsics interface {
	// GroupByDay is documented at Points.GroupByDay.
	GroupByDay(p Points, drift int64, align bool, location *time.Location) Group
	// GroupByYear is documented at Points.GroupByYear.
	GroupByYear(p Points, drift int64, align bool, location *time.Location) Group
	// GroupByMonth is documented at Points.GroupByMonth.
	GroupByMonth(p Points, drift int64, align bool, location *time.Location) Group
	// Scale is documented at Points.Scale.
	Scale(p Points, x, y int64) Points
	// Limit is documented at Points.Limit.
	Limit(p Points, min, max int64) Points
	// SnapToGrid is documented at Points.SnapToGrid.
	SnapToGrid(p Points, divisor int64) Points
	// PointsReduce is documented at Points.Reduce.
	PointsReduce(p Points, f AggregateFunc) (int64, bool)
	// M4 is documented at Points.M4.
	M4(p Points, width int64) Points
	// GroupReduce is documented at Group.Reduce.
	GroupReduce(g Group, f AggregateFunc) Points
	// GroupReduceTransposed is documented at Group.ReduceTransposed.
	GroupReduceTransposed(g Group, f AggregateFunc) Points
}

type mathStub struct {
}

func (m mathStub) PointsReduce(p Points, f AggregateFunc) (int64, bool) {
	return 0, false
}

func (m mathStub) GroupReduce(g Group, f AggregateFunc) Points {
	return Points{}
}

func (m mathStub) GroupReduceTransposed(g Group, f AggregateFunc) Points {
	return Points{}
}

func (m mathStub) Limit(p Points, min, max int64) Points {
	return Points{}
}

func (m mathStub) SnapToGrid(p Points, divisor int64) Points {
	return Points{}
}

func (m mathStub) AvgY(p Points) (int64, bool) {
	return 0, false
}

func (m mathStub) SumY(p Points) (int64, bool) {
	return 0, false
}

func (m mathStub) MinY(p Points) (int64, bool) {
	return 0, false
}

func (m mathStub) MaxY(p Points) (int64, bool) {
	return 0, false
}

func (m mathStub) M4(p Points, width int64) Points {
	return Points{}
}

func (m mathStub) GroupByYear(p Points, drift int64, align bool, location *time.Location) Group {
	return Group{}
}

func (m mathStub) GroupByDay(p Points, drift int64, align bool, location *time.Location) Group {
	return Group{}
}

func (m mathStub) GroupByMonth(p Points, drift int64, align bool, location *time.Location) Group {
	return Group{}
}

func (m mathStub) Scale(p Points, x, y int64) Points {
	return Points{}
}
