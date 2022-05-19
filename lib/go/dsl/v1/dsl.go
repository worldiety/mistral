// SPDX-FileCopyrightText: © 2022 The mistral authors <github.com/worldiety/mistral.git/lib/go/dsl/AUTHORS>
// SPDX-License-Identifier: BSD-2-Clause

package miel

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"golang.org/x/text/language"
)

// Evaluator specifies t^he func type for the Start callback which performs the actual query.
type Evaluator func(ctx context.Context)

// A ProcBuilder describes and configures a MiEl proc macro for later execution.
type ProcBuilder interface {
	// Parameter defines a function callback to return input and output/result parameter of this proc.
	// The concrete instances are used to provide example values to render.
	Parameter(func() (interface{}, interface{})) ProcBuilder

	// Start configures the given function to be executed for the evaluation.
	// Generally, a function must be thread safe to be invoked multiple times
	// concurrently.
	Start(Evaluator)
}

// Configure creates a ProcBuilder instance which depends on the execution environment.
func Configure() ProcBuilder {
	return &stubBuilder{}
}

// A TZ represents an unparsed IANA time zone and can be converted into a time.Location to perform calculations.
// Please note that this is not an offset. A time zone refers to a concrete place on earth and describes a very complex
// mapping to decide how to display a UTC date in a human-readable way. An offset will change randomly according to
// daylight saving times and politics in that concrete place. So, technically a time zone consists of an arbitrary
// amount of offsets and rules when to apply them.
type TZ string

// UnmarshalJSON provides just a json unmarshal serialization to validate the input.
func (t *TZ) UnmarshalJSON(bytes []byte) error {
	s := string(bytes)
	if _, err := time.LoadLocation(s); err != nil {
		return err
	}

	*t = TZ(s)
	return nil
}

// MustParse returns the actual Location or panics.
func (t TZ) MustParse() *time.Location {
	loc, err := time.LoadLocation(string(t))
	if err != nil {
		panic(err)
	}

	return loc
}

// Parse returns the parsed IANA time zone.
func (t TZ) Parse() (*time.Location, error) {
	loc, err := time.LoadLocation(string(t))
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", t, err)
	}

	return loc, nil
}

// Query unpacks the context specific DB.
func Query(ctx context.Context) DB {
	db := ctx.Value(ctxDB).(DB)
	if db == nil {
		panic("context does not contain a DB")
	}

	return db
}

// DB describes the contract to the Mistral database and provides a bunch of query methods.
type DB interface {
	// Bucket loads the metadata about the bucket which usually represents a device which generates a bunch of
	// time series data. Returns false if no such bucket exist. Panics for any other failure.
	Bucket(id UUID) (Bucket, bool)

	// Metric loads the metric metadata and describes a specific time series data which is required to interpret
	// the meaning of x and y values. Returns false if no such metric exist. Panics for any other failure.
	Metric(id UUID) (Metric, bool)

	// ScaleOf returns the scale for the given metric ID or returns 1 if not found. A multiple of 10,
	// usually in the range of 1, 10, 100 or 1000.
	ScaleOf(metricID UUID) int64

	// FindRanges returns all metrics which have at least a single data point and therefore represents a kind of
	// coverage. If multiple buckets (devices) have
	// the same metric, the overall min/max keys are determined and returned. The returned ranges are sorted by metric
	// id.
	FindRanges(bucketIDs []UUID) []DataRange

	// MinMax returns the minimum and maximum timestamp for the given metric within the denoted bucket (device).
	MinMax(bucketID, metricID UUID) DataRange

	// FindInRange loads those (time) series of the given buckets identified by the metric id, which exists.
	FindInRange(bucketIDs []UUID, metricID UUID, r Interval) Group
}

type translatedEntity interface {
	String() string
	LanguageTags() []string
	Translated() map[string]Translation
}

// translateName is a helper to match and return the translated Name of a Bucket or Metric. This is convenience
// helper to deliver a ready-to-use chart legend or axis description.
func translateName(ctx context.Context, translated translatedEntity) string {
	tag := MatchLanguage(ctx, translated.LanguageTags()...)
	if tag == "" {
		return translated.String()
	}

	return translated.Translated()[tag].Name
}

func translateNames(ctx context.Context, ids []UUID, f func(UUID) (translatedEntity, bool)) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		info, exists := f(id)
		if !exists {
			names = append(names, id.String())
		}

		name := translateName(ctx, info)
		names = append(names, name)
	}

	return names
}

// BucketNames translates the given buckets identified by their identifiers, if possible. If no translation exists,
// the default name is returned. If no metadata is available, the string representation of the ID is returned.
func BucketNames(ctx context.Context, bucketIDs []UUID) []string {
	db := Query(ctx)
	return translateNames(ctx, bucketIDs, func(uuid UUID) (translatedEntity, bool) {
		return db.Bucket(uuid)
	})
}

// MetricNames translates the given metrics identified by their identifiers, if possible. If no translation exists,
// the default name is returned. If no metadata is available, the string representation of the ID is returned.
func MetricNames(ctx context.Context, metricIDs []UUID) []string {
	db := Query(ctx)
	return translateNames(ctx, metricIDs, func(uuid UUID) (translatedEntity, bool) {
		return db.Metric(uuid)
	})
}

// Request parses the body from the given context into the given pointer. Panics for illegal arguments.
// Currently, supported are application/json and application/xml. Subsequent calls are undefined.
func Request(ctx context.Context, v interface{}) {
	req := mustRequest(ctx)

	switch req.Header.Get(contentType) {
	case mimeTypeJSON:
		dec := json.NewDecoder(req.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			panic(httpError{msg: "cannot decode json from body", status: http.StatusBadRequest, cause: err})
		}
	case mimeTypeXML:
		dec := xml.NewDecoder(req.Body)
		if err := dec.Decode(v); err != nil {
			panic(httpError{msg: "cannot decode xml from body", status: http.StatusBadRequest, cause: err})
		}
	default:
		panic(httpError{msg: "unsupported Content-Type: " + req.Header.Get(contentType), status: http.StatusBadRequest})
	}
}

// Response marshals the given value as json. If the first field has a xml-tag, the response is treated as
// xml. Panics for illegal arguments. Subsequent calls are undefined.
func Response(ctx context.Context, v interface{}) {
	w := ctx.Value(ctxHttpResponseWriter).(http.ResponseWriter)
	if w == nil {
		panic("context does not contain a http.ResponseWriter")
	}

	if typ := reflect.TypeOf(v); typ.NumField() > 0 {
		if _, isXML := typ.Field(0).Tag.Lookup("xml"); isXML {
			w.Header().Set(contentType, mimeTypeXML)
			enc := xml.NewEncoder(w)
			if err := enc.Encode(v); err != nil {
				panic(fmt.Errorf("cannot encode xml response: %w", err))
			}

			if err := enc.Flush(); err != nil {
				panic(fmt.Errorf("cannot flush xml response: %w", err))
			}

			return
		}
	}

	w.Header().Set(contentType, mimeTypeJSON)
	enc := json.NewEncoder(w)
	if err := enc.Encode(v); err != nil {
		panic(fmt.Errorf("cannot encode json response: %w", err))
	}
}

// MatchLanguage inspects the request (Accept-Language) and context and matches that against the given language
// IETF BCP 47 tags (like en, en-US, es-419 or az-Arab). If a tag cannot be parsed a panic is thrown.
// If the given tags and the required tag cannot be matched, the first tag is the default and returned.
// If no tags are given, the empty string is returned.
//
// See also https://go.dev/blog/matchlang to get a summary
// about the topic and the currently used underlying implementation.
func MatchLanguage(ctx context.Context, languageTags ...string) string {
	if len(languageTags) == 0 {
		return ""
	}

	r := mustRequest(ctx)

	tags := make([]language.Tag, 0, len(languageTags))
	for _, tag := range languageTags {
		tags = append(tags, language.MustParse(tag))
	}

	matcher := language.NewMatcher(tags)
	accept := r.Header.Get("Accept-Language")
	_, idx := language.MatchStrings(matcher, accept)

	return languageTags[idx]
}

// ViewportWidth resolves a hint for down sampling a data series suited for displaying data within a chart.
// The default is 512 and can be overridden by setting the Viewport-Width http header.
func ViewportWidth(ctx context.Context) int64 {
	r := mustRequest(ctx)
	v := r.Header.Get("Viewport-Width")
	if v == "" {
		return 512
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		panic(httpError{msg: "invalid Viewport-Width: ", status: http.StatusBadRequest, cause: err})
	}

	return i
}

// Timezone resolves the IANA timezone and location in the following order:
//  * UTC
//  * Takes the http request param X-TZ which may contain an IANA timezone
// See also the TZ type which can be used to transport and parse IANA timezone information.
// The purpose is that the request can set the time zone for calculations, especially for grouping by day, month or year
// which depends on the customers (tax) location. Intentionally the server location is not the default,
// because its location is not related to the data it is processing, especially when moving between cloud data centers.
//
// Please keep in mind, that offsets are not time zones.
func Timezone(ctx context.Context) *time.Location {
	r := mustRequest(ctx)
	tz := r.Header.Get("X-TZ")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		panic(httpError{msg: "invalid X-TZ location: " + tz, status: http.StatusBadRequest, cause: err})
	}

	return loc
}

// Time returns a helper instance located into the given Timezone as resolved by Timezone.
// If ctx is nil, a UTC zoned Times is returned.
func Time(ctx context.Context) Times {
	if ctx == nil {
		return Times{tz: time.UTC}
	}

	tz := Timezone(ctx)
	return Times{tz: tz}
}

// Times provides access to a variety of UTC and Interval operations based on a Timezone.
type Times struct {
	tz *time.Location
}

// Now returns the current time instant interpreted in the given time zone.
func (t Times) Now() time.Time {
	return time.Now().In(t.tz)
}

// ThisYear returns the UTC Interval in seconds from 01.01.20xx and 31.12.20xx based on the current Location.
func (t Times) ThisYear() Interval {
	return t.Year(t.Now().Year())
}

// Year returns the first UTC value in the given time zone for the given year and the according last UTC value.
func (t Times) Year(year int) Interval {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, t.tz)
	end := start.AddDate(1, 0, 0).Add(-time.Nanosecond)

	return Interval{
		Min: start.Unix(),
		Max: end.Unix(),
	}
}

// Today returns the first UTC value in the given time zone for the current day and the according last UTC value.
func (t Times) Today() Interval {
	return t.DayOf(0)
}

// DayOf returns the time zone interpreted UTC value with the given offset, where 0 means today and -1 yesterday.
func (t Times) DayOf(offset int) Interval {
	y, m, d := t.Now().Date()

	start := time.Date(y, m, d+offset, 0, 0, 0, 0, t.tz)
	end := time.Date(y, m, d+offset, 23, 59, 59, int(time.Second-time.Nanosecond), t.tz)

	return Interval{
		Min: start.Unix(),
		Max: end.Unix(),
	}
}

// A DataRange defines a metric id and the range of min/max x data it provides. Usually in Seconds since
// Unix Epoch. The meaning of ID is undefined and may be the zero UUID or refer to a bucket or metric or
// a bucket specific metric series. Inspect the according documentation of the exact method which
// creates such DataRange.
type DataRange struct {
	ID    UUID
	MinX  int64
	MaxX  int64
	Valid bool
}

// Interval contains the min and max unix timestamps, which have always 'inclusive' semantics. Usually in Seconds since
// Unix Epoch. See also Range, TZ and Timezone.
type Interval struct {
	Min, Max int64
}

// Metric describes a (time) series with a specific ID and the same semantics across Buckets (devices). For example,
// in the context of renewable energy a wind turbine has a bunch of metrics like production in kW, wind speed in km/h or
// wind direction in radians.
type Metric struct {
	ID           UUID                   `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Scale        int64                  `json:"scale"`
	Resolution   time.Duration          `json:"resolution"`
	Translations map[string]Translation `json:"translations"`
}

// LanguageTags returns an alphabetically sorted list of available translations. See also TranslateName.
func (m Metric) LanguageTags() []string {
	return sortedLangTags(m.Translations)
}

// Translated is a getter for Translations.
func (m Metric) Translated() map[string]Translation {
	return m.Translations
}

// String returns the name.
func (m Metric) String() string {
	return m.Name
}

// A Translation model helps to translate a Name and Description tuple into a specific language.
// See also MatchLanguage.
type Translation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// A Bucket represents an abstract namespace for a domain object containing unique (time) series data related
// to a specific metric. Usually, a Bucket represents a physical device generating data like a wind turbine.
// Therefore, an attached time zone and an individual name and description makes usually sense. However, it may
// also contain other virtual metrics like calculated business data for a customer (time zone may be tax related
// then).
type Bucket struct {
	ID           UUID                   `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Timezone     string                 `json:"timezone"`
	Translations map[string]Translation `json:"translations"`
}

// LanguageTags returns an alphabetically sorted list of available translations.
// See also MatchLanguage.
func (b Bucket) LanguageTags() []string {
	return sortedLangTags(b.Translations)
}

// Translated is a getter for Translations.
func (b Bucket) Translated() map[string]Translation {
	return b.Translations
}

// String returns the name.
func (b Bucket) String() string {
	return b.Name
}

// UUIDs is just a slice of UUID with some helper methods attached to write more readable implementations.
type UUIDs []UUID

// First returns the first UUID or panics.
func (s UUIDs) First() UUID {
	return s[0]
}

// Join concat the other UUIDs to this slice of UUIDs and returns the new slice.
func (s UUIDs) Join(other UUIDs) UUIDs {
	return append(s, other...)
}

// Point represents a packed and optimized data point which is usually part of a larger time series represented as
// Points.
type Point struct {
	// X is usually in Seconds since Unix Epoch.
	X int64 `json:"x"`
	// Y is usually a pre-scaled decimal metric value. Use ScaleOf to post-multiply to get
	Y int64 `json:"y"`
}

// FPoint is usually used for serialization into JSON and to be ready to be displayed by
// consumer agents, e.g. written in Java or JavaScript.
// Intentionally this type does not provide any built-in operations
// because it should be the last step of processing, if not avoidable at all. Floating point numbers should only
// be used for display purposes and not for calculations, to avoid rounding errors which become significant
// when calculating with billions of numbers.
type FPoint struct {
	// X is usually in milliseconds since Unix Epoch.
	X int64 `json:"x"`

	// Y is usually already un-pre-multiplied and ready to display.
	Y float64 `json:"y"`
}

// FPoints is just a slice of FPoint elements. See also FPoint.
type FPoints []FPoint

// FGroup is just a slice of FPoints elements.
type FGroup []FPoints

// Join appends the given other series to this series and returns the new slice.
func (p FGroup) Join(other FGroup) FGroup {
	return append(p, other...)
}

// NoDrift represents the 0 literal to improve readability. Usually in Seconds.
// The drift value is added to each timestamp, so that a drift of the points can be
// respected (e.g. due to start- or end aggregated data points).
const NoDrift = 0

// AlignGroupStart represents the true literal to improve readability.
// For the Group* functions, the parameter determines if the natural start of the grouping is set to all X values for
// each group (first unix time stamp of the group start at 00:00:00) after applying the drift.
const AlignGroupStart = true

// DefaultGrid is 600 seconds.
const DefaultGrid = 600

// Points is just a slice of Point with a bunch of optimized helper methods for data analysis.
type Points []Point

// GroupByDay takes all points and interprets the Point.X value as a unix timestamp in seconds. The shift value is
// added to each timestamp, so that a drift of the points can be respected (e.g. due to start- or end aggregated
// data points). If parameter raster is true, the natural start of the grouping is set to all X values for each
// group (first unix time stamp of the day at 00:00:00) after applying the shift.
//
// It expects that points are ordered ascended by X (==time). The result is undefined, if the dataset is not
// sorted correctly. Location may not be nil.
func (p Points) GroupByDay(drift int64, align bool, location *time.Location) Group {
	return Math.GroupByDay(p, drift, align, location)
}

// GroupByYear takes all points and interprets the Point.X value as a unix timestamp in seconds. The shift value is
// added to each timestamp, so that a drift of the points can be respected (e.g. due to start- or end aggregated
// data points). If parameter raster is true, the natural start of the grouping is set to all X values for each
// group (first unix time stamp of the year, 1. January 00:00:00) after applying the shift.
//
// It expects that points are ordered ascended by X (==time). The result is undefined, if the dataset is not
// sorted correctly. Location may not be nil.
func (p Points) GroupByYear(drift int64, align bool, location *time.Location) Group {
	return Math.GroupByYear(p, drift, align, location)
}

// GroupByMonth takes all points and interprets the Point.X value as a unix timestamp in seconds. The shift value is
// added to each timestamp, so that a drift of the points can be respected (e.g. due to start- or end aggregated
// data points). If parameter raster is true, the natural start of the grouping is set to all X values for each
// group (first unix time stamp of the month, at the first day at 00:00:00) after applying the shift.
//
// It expects that points are ordered ascended by X (==time). The result is undefined, if the dataset is not
// sorted correctly. Location may not be nil.
func (p Points) GroupByMonth(drift int64, align bool, location *time.Location) Group {
	return Math.GroupByMonth(p, drift, align, location)
}

// Scale multiplies all points within the series with the given x,y scalars.
func (p Points) Scale(x, y int64) Points {
	return Math.Scale(p, x, y)
}

// Unscale multiplies the X value by 1000 to get Milliseconds and divides by yScale using floating point arithmetics.
// This should be the last step after Downsample and performs another allocation.
func (p Points) Unscale(yScale int64) FPoints {
	res := make(FPoints, 0, len(p))
	for _, point := range p {
		res = append(res, FPoint{
			X: point.X * 1000,
			Y: float64(point.Y) / float64(yScale),
		})
	}

	return res
}

// Limit mutates pts so that it only contains y-values which are larger than min and smaller than max (inclusive).
func (p Points) Limit(min, max int64) Points {
	return Math.Limit(p, min, max)
}

// SnapToGrid divides by divisor and multiplies back, causing the according truncation. Example rasterization with
// a divisor of 600:
//  *  300 =>    0
//  *  601 =>  600
//  * 1202 => 1200
//  * 1700 => 1200
func (p Points) SnapToGrid(divisor int64) Points {
	return Math.SnapToGrid(p, divisor)
}

// Reduce applies the given AggregateFunc and returns the result or false, if the value cannot be
// calculated. In general, points cannot be reduced, if no values are available like calculating
// an average which would cause a divide by zero error.
func (p Points) Reduce(f AggregateFunc) (int64, bool) {
	return Math.PointsReduce(p, f)
}

// Last returns the last Point.
func (p Points) Last() (Point, bool) {
	if len(p) == 0 {
		return Point{}, false
	}

	return p[len(p)-1], true
}

// First returns the first Point.
func (p Points) First() (Point, bool) {
	if len(p) == 0 {
		return Point{}, false
	}

	return p[0], true
}

// Downscale discards points which are insignificant when displaying in the given width.
// This uses the default downscale implementation, which may change between revisions to
// optimize experience. Width should be the amount of pixel on which a line chart should be drawn.
// See also M4 which is currently used.
func (p Points) Downscale(width int64) Points {
	return p.M4(width)
}

// M4 applies the according downscaling algorithm for visualization by Uwe Jugel, Zbigniew Jerzak,
// Gregor Hackenbroich and Volker Markl. See http://www.vldb.org/pvldb/vol7/p797-jugel.pdf. It expects
// that the given db.TimeSeries is already sorted.
//
// The width determines how many buckets are created in the given time interval as defined by the time series.
// Each bucket may have a variable amount of entries, which are sampled to at most 4 values:
// the highest/lowest values and the max/min values. If these points overlap, they are only returned once, so
// at worst only one value per bucket is returned.
//
// If the width is larger than the amount of available points, the original points are returned.
func (p Points) M4(width int64) Points {
	return Math.M4(p, width)
}

// AggregateFunc is an enum like type to identify an aggregate function for Group.Reduce or Group.ReduceTransposed
// functions.
type AggregateFunc int

// Valid determines if AggregateFunc defines a valid enum.
// See also MinY, MaxY, AvgY, SumY and Count.
func (f AggregateFunc) Valid() bool {
	return f >= MinY && f <= Count
}

const (
	// MinY returns the minimum Y value.
	MinY AggregateFunc = iota + 1

	// MaxY returns the maximum Y value.
	MaxY

	// AvgY sums all Y values up and performs a float64 division with rounding.
	AvgY

	// SumY returns the sum of all Y values.
	SumY

	// Count returns the amount of entries.
	Count
)

// Group is slice of (time) series points.
type Group []Points

// First returns the first series or panics.
func (p Group) First() Points {
	return p[0]
}

// Reduce applies an AggregateFunc on the group and returns a single series again.
// Technically it inner loops over each group on an "as is" basis. Example:
//  [
//    [(1|2), (2|3), (4|5)],
//    [(5|6), (7|8), (9|10)],
//    [(11|12), (13|14), (15|16)]
//  ]
//  => f is called as follows:
//    [(1|2), (2|3), (4|5)]
//    [(5|6), (7|8), (9|10)]
//    [(11|12), (13|14), (15|16)]
// Note that the X value if always the first of each group.
// Also note, that this is weired for Max, because it returns the "wrong" x (the first, as defined).
func (p Group) Reduce(f AggregateFunc) Points {
	return Math.GroupReduce(p, f)
}

// ReduceTransposed applies an AggregateFunc on the group and returns a single series again.
// OuterGroupByX transposes the points of each group into a new artificial group and invokes f on it. It requires that
// each group is sorted ascending by X. The result on unsorted groups is undefined. Example:
//  [
//    [(1|2), (2|3), (4|5)],
//    [(1|6), (2|8), (8|10)],
//    [(0|12), (2|14), (4|16)]
//  ]
//  => f is called as follows:
//    [(0|12)],
//    [(1|2), (1|6)],
//    [(2|3), (2|8), (2|14)]
//    [(4|5), (4|16)]
//    [(8|10]
func (p Group) ReduceTransposed(f AggregateFunc) Points {
	return Math.GroupReduceTransposed(p, f)
}

// ForEach allows an in-line modification of each point series inside Group. For example
// one can SnapToGrid, then create a GroupByDay aggregation with a reduction into a series by using the AvgY operator.
// A lot of operations can be applied in-place to reduce memory footprint and pressure. See also ForEachF.
func (p Group) ForEach(f func(pts Points) Points) Group {
	for i, points := range p {
		p[i] = f(points)
	}

	return p
}

// ForEachF is like ForEach but allows a transformation into a floating point series resulting in a floating point
// group of series. It is guaranteed that the transformation performs additional heap allocations and therefore
// should only be used after Downsampling.
func (p Group) ForEachF(f func(pts Points) FPoints) FGroup {
	res := make(FGroup, 0, len(p))
	for _, points := range p {
		res = append(res, f(points))
	}

	return res
}
