package handlers

import (
	"fmt"
	"strings"
)

// PrometheusFormatter formats metrics in Prometheus format
type PrometheusFormatter struct{}

// NewPrometheusFormatter creates a new Prometheus formatter
func NewPrometheusFormatter() *PrometheusFormatter {
	return &PrometheusFormatter{}
}

// Format formats metrics data into Prometheus text format
func (p *PrometheusFormatter) Format(data *MetricsData) string {
	var result strings.Builder

	// Format database metrics
	for _, metric := range data.DatabaseMetrics {
		result.WriteString(p.formatMetric(metric))
		result.WriteString("\n")
	}

	// Format application metrics
	for _, metric := range data.AppMetrics {
		result.WriteString(p.formatMetric(metric))
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// formatMetric formats a single metric
func (p *PrometheusFormatter) formatMetric(metric Metric) string {
	var result strings.Builder

	// Add HELP comment if provided
	if metric.Help != "" {
		result.WriteString(fmt.Sprintf("# HELP %s %s\n", metric.Name, metric.Help))
	}

	// Add TYPE comment (assuming gauge for simplicity)
	result.WriteString(fmt.Sprintf("# TYPE %s gauge\n", metric.Name))

	// Format labels
	labelStr := p.formatLabels(metric.Labels)

	// Add metric line
	result.WriteString(fmt.Sprintf("%s%s %s", metric.Name, labelStr, p.formatValue(metric.Value)))

	return result.String()
}

// formatLabels formats metric labels
func (p *PrometheusFormatter) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var pairs []string
	for key, value := range labels {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, key, value))
	}

	return fmt.Sprintf("{%s}", strings.Join(pairs, ","))
}

// formatValue formats metric values (integers as integers, floats as floats)
func (p *PrometheusFormatter) formatValue(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%g", value)
}