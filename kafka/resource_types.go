package kafka

import (
	"fmt"

	"github.com/twmb/franz-go/pkg/kmsg"
)

// ConfigResourceType represents the type of Kafka resource for configuration operations
type ConfigResourceType int

// ConfigResourceType constants - matching kmsg.ConfigResourceType values
const (
	ConfigResourceTypeUnknown     ConfigResourceType = 0
	ConfigResourceTypeTopic       ConfigResourceType = 2
	ConfigResourceTypeBroker      ConfigResourceType = 4
	ConfigResourceTypeGroup       ConfigResourceType = 3
	ConfigResourceTypeUser        ConfigResourceType = 6
	ConfigResourceTypeClientQuota ConfigResourceType = 7
)

// StringToResourceType converts a string resource type to its corresponding ConfigResourceType
func StringToResourceType(resourceTypeStr string) (ConfigResourceType, error) {
	switch resourceTypeStr {
	case "topic":
		return ConfigResourceTypeTopic, nil
	case "broker":
		return ConfigResourceTypeBroker, nil
	case "group":
		return ConfigResourceTypeGroup, nil
	case "user":
		return ConfigResourceTypeUser, nil
	case "client_quota":
		return ConfigResourceTypeClientQuota, nil
	default:
		return ConfigResourceTypeUnknown, fmt.Errorf("unknown resource type: %s", resourceTypeStr)
	}
}

// ToKmsgResourceType converts internal ConfigResourceType to franz-go kmsg.ConfigResourceType
// This keeps franz-go dependency isolated to the kafka package
func (rt ConfigResourceType) ToKmsgResourceType() kmsg.ConfigResourceType {
	return kmsg.ConfigResourceType(rt)
}

// String returns the string representation of the resource type
func (rt ConfigResourceType) String() string {
	switch rt {
	case ConfigResourceTypeTopic:
		return "topic"
	case ConfigResourceTypeBroker:
		return "broker"
	case ConfigResourceTypeGroup:
		return "group"
	case ConfigResourceTypeUser:
		return "user"
	case ConfigResourceTypeClientQuota:
		return "client_quota"
	default:
		return "unknown"
	}
}
