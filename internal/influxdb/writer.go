package influxdb

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.uber.org/zap"

	"github.com/anker-solix-exporter/anker-solix-exporter/internal/anker"
)

type Writer struct {
	client      influxdb2.Client
	writeAPI    api.WriteAPIBlocking
	org         string
	bucket      string
	measurement string
	logger      *zap.Logger
}

func NewWriter(url, token, org, bucket, measurement string, logger *zap.Logger) (*Writer, error) {
	client := influxdb2.NewClient(url, token)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	health, err := client.Health(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		client.Close()
		msg := ""
		if health.Message != nil {
			msg = *health.Message
		}
		return nil, fmt.Errorf("InfluxDB health check failed: %s", msg)
	}

	logger.Info("connected to InfluxDB",
		zap.String("url", url),
		zap.String("org", org),
		zap.String("bucket", bucket),
	)

	return &Writer{
		client:      client,
		writeAPI:    client.WriteAPIBlocking(org, bucket),
		org:         org,
		bucket:      bucket,
		measurement: measurement,
		logger:      logger,
	}, nil
}

func (w *Writer) WriteMeasurements(ctx context.Context, measurements []anker.Measurement) error {
	if len(measurements) == 0 {
		return nil
	}

	for _, m := range measurements {
		point := influxdb2.NewPoint(
			w.measurement,
			map[string]string{
				"site_id":     m.SiteID,
				"site_name":   m.SiteName,
				"device_sn":   m.DeviceSN,
				"device_name": m.DeviceName,
				"device_type": m.DeviceType,
			},
			map[string]interface{}{
				"solar_power":   m.SolarPower,
				"output_power":  m.OutputPower,
				"grid_power":    m.GridPower,
				"battery_power": m.BatteryPower,
				"battery_soc":   m.BatterySoC,
			},
			m.Timestamp,
		)
		
		if err := w.writeAPI.WritePoint(ctx, point); err != nil {
			return fmt.Errorf("failed to write point to InfluxDB: %w", err)
		}
	}

	w.logger.Debug("wrote measurements to InfluxDB",
		zap.Int("count", len(measurements)),
	)

	return nil
}

func (w *Writer) GetLastTimestamp(ctx context.Context, siteID, deviceSN string) (time.Time, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: -90d)
		  |> filter(fn: (r) => r["_measurement"] == "%s")
		  |> filter(fn: (r) => r["site_id"] == "%s")
		  |> filter(fn: (r) => r["device_sn"] == "%s")
		  |> last()
		  |> keep(columns: ["_time"])
	`, w.bucket, w.measurement, siteID, deviceSN)

	queryAPI := w.client.QueryAPI(w.org)
	result, err := queryAPI.Query(ctx, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("query failed: %w", err)
	}
	defer result.Close()

	if result.Next() {
		if t, ok := result.Record().Time().MarshalText(); ok == nil {
			var timestamp time.Time
			if err := timestamp.UnmarshalText(t); err == nil {
				return timestamp, nil
			}
		}
		return result.Record().Time(), nil
	}

	if result.Err() != nil {
		return time.Time{}, fmt.Errorf("query error: %w", result.Err())
	}

	// No data found
	return time.Time{}, nil
}

func (w *Writer) Close() {
	if w.client != nil {
		w.client.Close()
	}
}
