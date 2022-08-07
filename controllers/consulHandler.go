package controllers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	consul "github.com/hashicorp/consul/api"
)

const consulClientSpawnError = "Failed to connect to consul server. Please check auth and consul address"

// This will come from env variable in the operator deployment.
var consulAddress = os.Getenv("CONSUL_HOST") + ":" + os.Getenv("CONSUL_PORT")

const pathPrefix = "/"

// SyncKeyValueToConsul selectively syncs drifted keys. If consul is in Japan region, it can solve Tokyo drift
func SyncKeyValueToConsul(key string, value string, reqLogger logr.Logger) {
	client, err := consul.NewClient(&consul.Config{Address: consulAddress})
	if err != nil {
		reqLogger.Error(err, consulClientSpawnError)
	}
	p := consul.KVPair{Key: key, Value: []byte(value)}
	val, _, err := client.KV().Get(key, nil)
	if err != nil {
		reqLogger.Error(err, "Failed to get existing value for key "+key+". Populating a new value")
	}

	if val != nil {
		if string(val.Value) != value {
			reqLogger.Info("Drift in configuration for key: " + key)
			reqLogger.Info("Value in consul: " + string(val.Value) + ". Desired state: " + value)
			reqLogger.Info("Reconciling state in consul server")
			_, err = client.KV().Put(&p, nil)
			if err != nil {
				reqLogger.Error(err, "Failed to update state in consul. Please check consul server uptime and auth")
			}
			reqLogger.Info("Task Failed Successfully.")
		}
	} else {
		_, err = client.KV().Put(&p, nil)
		if err != nil {
			reqLogger.Error(err, "Failed to update state in consul. Please check consul server uptime and auth")
		}
	}
}

// DeleteKeysFromConsul does what it says.
func DeleteKeysFromConsul(key string, reqLogger logr.Logger) {
	client, err := consul.NewClient(&consul.Config{Address: consulAddress})
	if err != nil {
		reqLogger.Error(err, consulClientSpawnError)
	}
	_, err = client.KV().Delete(key, nil)
	if err != nil {
		reqLogger.Error(err, "Failed to delete key in consul. Key: "+key)
	}
}

// FlattenNestedKeysToPathInterface converts an interface map to flat interface map.
// Do not delete, this function may be used. Optimization is not needed here
func FlattenNestedKeysToPathInterface(prefix string, src map[interface{}]interface{}, dest map[interface{}]interface{}) {
	if len(prefix) > 0 {
		prefix += pathPrefix
	}
	for k, v := range src {
		switch child := v.(type) {
		case map[interface{}]interface{}:
			FlattenNestedKeysToPathInterface(prefix+fmt.Sprintf("%v", k), child, dest)
		case []interface{}:
			for i := 0; i < len(child); i++ {
				dest[prefix+fmt.Sprintf("%v", k)+"."+strconv.Itoa(i)] = child[i]
			}
		default:
			dest[prefix+fmt.Sprintf("%v", k)] = v
		}
	}
}

// FlattenNestedKeysToPath converts an interface map to flat string map.
// Both functions are different. Do not try to merge
func FlattenNestedKeysToPath(prefix string, src map[interface{}]interface{}, dest map[string]string) {
	if len(prefix) > 0 {
		prefix += pathPrefix
	}
	for k, v := range src {
		switch child := v.(type) {
		case map[interface{}]interface{}:
			FlattenNestedKeysToPath(prefix+fmt.Sprintf("%v", k), child, dest)
		case []interface{}:
			for i := 0; i < len(child); i++ {
				dest[prefix+fmt.Sprintf("%v", k)+"."+strconv.Itoa(i)] = fmt.Sprintf("%v", child[i])
			}
		default:
			dest[prefix+fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
	}
}
