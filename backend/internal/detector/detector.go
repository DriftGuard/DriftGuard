package detector

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/DriftGuard/core/pkg/models"
	"go.uber.org/zap"
)

type DriftDetector struct {
	logger *zap.Logger
}

func NewDriftDetector(logger *zap.Logger) *DriftDetector {
	return &DriftDetector{
		logger: logger,
	}
}

func (d *DriftDetector) DetectDrift(
	resourceID string,
	liveState map[string]interface{},
	desiredState map[string]interface{},
) *models.DriftResult {
	d.logger.Debug("Starting drift detection", zap.String("resource_id", resourceID))

	result := &models.DriftResult{
		ResourceID:    resourceID,
		Detected:      false,
		Changes:       []models.DriftChange{},
		LastEvaluated: time.Now(),
		LiveState:     liveState,
		DesiredState:  desiredState,
	}

	changes := d.deepCompare("", liveState, desiredState)

	if len(changes) > 0 {
		result.Detected = true
		result.Changes = changes
		d.logger.Info("Drift detected",
			zap.String("resource_id", resourceID),
			zap.Int("changes_count", len(changes)))
	} else {
		d.logger.Debug("No drift detected", zap.String("resource_id", resourceID))
	}

	return result
}

func (d *DriftDetector) deepCompare(path string, live, desired interface{}) []models.DriftChange {
	var changes []models.DriftChange

	if reflect.TypeOf(live) != reflect.TypeOf(desired) {
		change := d.createDriftChange(path, desired, live, "TypeChange", "medium")
		changes = append(changes, change)
		return changes
	}

	switch v := live.(type) {
	case map[string]interface{}:
		desiredMap, ok := desired.(map[string]interface{})
		if !ok {
			return changes
		}

		for key, desiredValue := range desiredMap {
			currentPath := key
			if path != "" {
				currentPath = fmt.Sprintf("%s.%s", path, key)
			}

			liveValue, exists := v[key]
			if !exists {
				change := d.createDriftChange(currentPath, desiredValue, nil, "MissingField", "high")
				changes = append(changes, change)
				continue
			}

			nestedChanges := d.deepCompare(currentPath, liveValue, desiredValue)
			changes = append(changes, nestedChanges...)
		}

	case []interface{}:
		desiredSlice, ok := desired.([]interface{})
		if !ok {
			return changes
		}

		if len(v) != len(desiredSlice) {
			change := d.createDriftChange(path, len(desiredSlice), len(v), "ArrayLength", "medium")
			changes = append(changes, change)
		}

		maxLen := len(v)
		if len(desiredSlice) > maxLen {
			maxLen = len(desiredSlice)
		}

		for i := 0; i < maxLen; i++ {
			currentPath := fmt.Sprintf("%s[%d]", path, i)

			if i < len(v) && i < len(desiredSlice) {
				nestedChanges := d.deepCompare(currentPath, v[i], desiredSlice[i])
				changes = append(changes, nestedChanges...)
			} else if i >= len(v) {
				change := d.createDriftChange(currentPath, desiredSlice[i], nil, "MissingElement", "medium")
				changes = append(changes, change)
			} else if i >= len(desiredSlice) {
				change := d.createDriftChange(currentPath, nil, v[i], "ExtraElement", "low")
				changes = append(changes, change)
			}
		}

	default:
		if !reflect.DeepEqual(live, desired) {
			change := d.createDriftChange(path, desired, live, d.classifyChange(path, desired, live), d.assessSeverity(path, desired, live))
			changes = append(changes, change)
		}
	}

	return changes
}

func (d *DriftDetector) createDriftChange(field string, from, to interface{}, changeType, severity string) models.DriftChange {
	return models.DriftChange{
		Field:    field,
		From:     from,
		To:       to,
		Type:     changeType,
		Severity: severity,
	}
}

func (d *DriftDetector) classifyChange(field string, from, to interface{}) string {
	fieldLower := strings.ToLower(field)

	switch {
	case strings.Contains(fieldLower, "replicas"):
		return "Scaling"
	case strings.Contains(fieldLower, "image"):
		return "VersionChange"
	case strings.Contains(fieldLower, "resources"):
		return "ResourceChange"
	case strings.Contains(fieldLower, "env"):
		return "ConfigurationChange"
	case strings.Contains(fieldLower, "labels"):
		return "LabelChange"
	case strings.Contains(fieldLower, "annotations"):
		return "AnnotationChange"
	case strings.Contains(fieldLower, "ports"):
		return "PortChange"
	case strings.Contains(fieldLower, "volume"):
		return "VolumeChange"
	case strings.Contains(fieldLower, "secret"):
		return "SecretChange"
	case strings.Contains(fieldLower, "configmap"):
		return "ConfigMapChange"
	default:
		return "GenericChange"
	}
}

func (d *DriftDetector) assessSeverity(field string, from, to interface{}) string {
	fieldLower := strings.ToLower(field)

	switch {
	case strings.Contains(fieldLower, "image"):
		return "high"
	case strings.Contains(fieldLower, "secret"):
		return "high"
	case strings.Contains(fieldLower, "resources.limits"):
		return "high"
	case strings.Contains(fieldLower, "replicas") && d.isSignificantReplicaChange(from, to):
		return "high"
	}

	switch {
	case strings.Contains(fieldLower, "replicas"):
		return "medium"
	case strings.Contains(fieldLower, "env"):
		return "medium"
	case strings.Contains(fieldLower, "ports"):
		return "medium"
	case strings.Contains(fieldLower, "volume"):
		return "medium"
	case strings.Contains(fieldLower, "resources.requests"):
		return "medium"
	}

	switch {
	case strings.Contains(fieldLower, "labels"):
		return "low"
	case strings.Contains(fieldLower, "annotations"):
		return "low"
	default:
		return "low"
	}
}

func (d *DriftDetector) isSignificantReplicaChange(from, to interface{}) bool {
	fromInt, fromOk := d.toInt(from)
	toInt, toOk := d.toInt(to)

	if !fromOk || !toOk {
		return false
	}

	diff := abs(fromInt - toInt)
	percentChange := float64(diff) / float64(fromInt) * 100

	return percentChange >= 50
}

func (d *DriftDetector) toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
