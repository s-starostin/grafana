package models

import (
	"context"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/pkg/components/null"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/models"
	"github.com/timberio/go-datemath"
)

// DataSubQuery represents a data sub-query.
type DataSubQuery struct {
	RefID         string             `json:"refId"`
	Model         *simplejson.Json   `json:"model,omitempty"`
	DataSource    *models.DataSource `json:"datasource"`
	MaxDataPoints int64              `json:"maxDataPoints"`
	IntervalMS    int64              `json:"intervalMs"`
	QueryType     string             `json:"queryType"`
}

// DataQuery contains all information about a data query request.
type DataQuery struct {
	TimeRange *DataTimeRange
	Queries   []DataSubQuery
	Headers   map[string]string
	Debug     bool
	User      *models.SignedInUser
}

type DataTimeRange struct {
	From string
	To   string
	now  time.Time
}

type TSDBTable struct {
	Columns []TSDBTableColumn `json:"columns"`
	Rows    []TSDBRowValues   `json:"rows"`
}

type TSDBTableColumn struct {
	Text string `json:"text"`
}

type TSDBTimePoint [2]null.Float
type TSDBTimeSeriesPoints []TSDBTimePoint
type TSDBTimeSeriesSlice []TSDBTimeSeries
type TSDBRowValues []interface{}

type DataQueryResult struct {
	Error       error               `json:"-"`
	ErrorString string              `json:"error,omitempty"`
	RefID       string              `json:"refId"`
	Meta        *simplejson.Json    `json:"meta,omitempty"`
	Series      TSDBTimeSeriesSlice `json:"series"`
	Tables      []TSDBTable         `json:"tables"`
	Dataframes  DataFrames          `json:"dataframes"`
}

type TSDBTimeSeries struct {
	Name   string               `json:"name"`
	Points TSDBTimeSeriesPoints `json:"points"`
	Tags   map[string]string    `json:"tags,omitempty"`
}

type DataResponse struct {
	Results map[string]DataQueryResult `json:"results"`
	Message string                     `json:"message,omitempty"`
}

type TSDBPlugin interface {
	DataQuery(ctx context.Context, ds *models.DataSource, query DataQuery) (DataResponse, error)
}

func NewDataTimeRange(from, to string) DataTimeRange {
	return DataTimeRange{
		From: from,
		To:   to,
		now:  time.Now(),
	}
}

func (tr *DataTimeRange) GetFromAsMsEpoch() int64 {
	return tr.MustGetFrom().UnixNano() / int64(time.Millisecond)
}

func (tr *DataTimeRange) GetFromAsSecondsEpoch() int64 {
	return tr.GetFromAsMsEpoch() / 1000
}

func (tr *DataTimeRange) GetFromAsTimeUTC() time.Time {
	return tr.MustGetFrom().UTC()
}

func (tr *DataTimeRange) GetToAsMsEpoch() int64 {
	return tr.MustGetTo().UnixNano() / int64(time.Millisecond)
}

func (tr *DataTimeRange) GetToAsSecondsEpoch() int64 {
	return tr.GetToAsMsEpoch() / 1000
}

func (tr *DataTimeRange) GetToAsTimeUTC() time.Time {
	return tr.MustGetTo().UTC()
}

func (tr *DataTimeRange) MustGetFrom() time.Time {
	res, err := tr.ParseFrom()
	if err != nil {
		return time.Unix(0, 0)
	}
	return res
}

func (tr *DataTimeRange) MustGetTo() time.Time {
	res, err := tr.ParseTo()
	if err != nil {
		return time.Unix(0, 0)
	}
	return res
}

func (tr DataTimeRange) ParseFrom() (time.Time, error) {
	return parseTimeRange(tr.From, tr.now, false, nil)
}

func (tr DataTimeRange) ParseTo() (time.Time, error) {
	return parseTimeRange(tr.To, tr.now, true, nil)
}

func (tr DataTimeRange) ParseFromWithLocation(location *time.Location) (time.Time, error) {
	return parseTimeRange(tr.From, tr.now, false, location)
}

func (tr DataTimeRange) ParseToWithLocation(location *time.Location) (time.Time, error) {
	return parseTimeRange(tr.To, tr.now, true, location)
}

func parseTimeRange(s string, now time.Time, withRoundUp bool, location *time.Location) (time.Time, error) {
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		seconds := val / 1000
		nano := (val - seconds*1000) * 1000000
		return time.Unix(seconds, nano), nil
	}

	diff, err := time.ParseDuration("-" + s)
	if err != nil {
		options := []func(*datemath.Options){
			datemath.WithNow(now),
			datemath.WithRoundUp(withRoundUp),
		}
		if location != nil {
			options = append(options, datemath.WithLocation(location))
		}

		return datemath.ParseAndEvaluate(s, options...)
	}

	return now.Add(diff), nil
}

// SeriesToFrame converts a TSDBTimeSeries to an SDK frame.
func SeriesToFrame(series TSDBTimeSeries) (*data.Frame, error) {
	timeVec := make([]*time.Time, len(series.Points))
	floatVec := make([]*float64, len(series.Points))
	for idx, point := range series.Points {
		timeVec[idx], floatVec[idx] = convertTSDBTimePoint(point)
	}
	frame := data.NewFrame(series.Name,
		data.NewField("time", nil, timeVec),
		data.NewField("value", data.Labels(series.Tags), floatVec),
	)

	return frame, nil
}

// convertTSDBTimePoint converts a TSDBTimePoint into two values appropriate
// for Series values.
func convertTSDBTimePoint(point TSDBTimePoint) (t *time.Time, f *float64) {
	timeIdx, valueIdx := 1, 0
	if point[timeIdx].Valid { // Assuming valid is null?
		tI := int64(point[timeIdx].Float64)
		uT := time.Unix(tI/int64(1e+3), (tI%int64(1e+3))*int64(1e+6)) // time.Time from millisecond unix ts
		t = &uT
	}
	if point[valueIdx].Valid {
		f = &point[valueIdx].Float64
	}
	return
}