// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusremotewriteexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/service/featuregate"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry"
)

// Config defines configuration for Remote Write exporter.
type Config struct {
	config.ExporterSettings        `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`
	sanitizeLabel                  bool

	// prefix attached to each exported metric name
	// See: https://prometheus.io/docs/practices/naming/#metric-names
	Namespace string `mapstructure:"namespace"`

	// QueueConfig allows users to fine tune the queues
	// that handle outgoing requests.
	RemoteWriteQueue RemoteWriteQueue `mapstructure:"remote_write_queue"`

	// MultiTenancy allows user to enable multi-tenancy support
	// from the exporter by setting http header or query param
	MultiTenancy MultiTenancy `mapstructure:"multi_tenancy"`

	// ExternalLabels defines a map of label keys and values that are allowed to start with reserved prefix "__"
	ExternalLabels map[string]string `mapstructure:"external_labels"`

	HTTPClientSettings confighttp.HTTPClientSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// ResourceToTelemetrySettings is the option for converting resource attributes to telemetry attributes.
	// "Enabled" - A boolean field to enable/disable this option. Default is `false`.
	// If enabled, all the resource attributes will be converted to metric labels by default.
	ResourceToTelemetrySettings resourcetotelemetry.Settings `mapstructure:"resource_to_telemetry_conversion"`
	WAL                         *WALConfig                   `mapstructure:"wal"`
}

// RemoteWriteQueue allows to configure the remote write queue.
type RemoteWriteQueue struct {
	// Enabled if false the queue is not enabled, the export requests
	// are executed synchronously.
	Enabled bool `mapstructure:"enabled"`

	// QueueSize is the maximum number of OTLP metric batches allowed
	// in the queue at a given time. Ignored if Enabled is false.
	QueueSize int `mapstructure:"queue_size"`

	// NumWorkers configures the number of workers used by
	// the collector to fan out remote write requests.
	NumConsumers int `mapstructure:"num_consumers"`
}

// MultiTenancy allows to configure multitenancy support.
type MultiTenancy struct {
	// Enabled if false multinenancy is not enabled.
	Enabled bool `mapstructure:"enabled"`

	// Header is the header name to set when sending metrics
	// to the backend.
	Header string `mapstructure:"header"`

	// QueryParam is the query parameter to set when sending metrics
	// to the backend.
	QueryParam string `mapstructure:"query_param"`

	// FromLabel uses this label value as tenant name.
	FromLabel string `mapstructure:"from_label"`

	// DefaultTenant is the default tenant name.
	DefaultTenant string `mapstructure:"default_tenant"`
}

var dropSanitizationGate = featuregate.Gate{
	ID:          "exporter.prometheusremotewrite.PermissiveLabelSanitization",
	Enabled:     false,
	Description: "Controls whether to change labels starting with '_' to 'key_'",
}

func init() {
	featuregate.GetRegistry().MustRegister(dropSanitizationGate)
}

// TODO(jbd): Add capacity, max_samples_per_send to QueueConfig.

var _ config.Exporter = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.RemoteWriteQueue.QueueSize < 0 {
		return fmt.Errorf("remote write queue size can't be negative")
	}

	if cfg.RemoteWriteQueue.Enabled && cfg.RemoteWriteQueue.QueueSize == 0 {
		return fmt.Errorf("a 0 size queue will drop all the data")
	}

	if cfg.RemoteWriteQueue.NumConsumers < 0 {
		return fmt.Errorf("remote write consumer number can't be negative")
	}

	if cfg.MultiTenancy.Enabled && cfg.MultiTenancy.Header == "" && cfg.MultiTenancy.QueryParam == "" {
		return fmt.Errorf("one of multi_tenancy header or query_param should be set")
	}

	if cfg.MultiTenancy.Enabled && cfg.MultiTenancy.FromLabel == "" {
		return fmt.Errorf("from_label should be set to find tenant name")
	}

	return nil
}
