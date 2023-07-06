package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/piyushsingariya/kaku/jsonschema"
	"github.com/piyushsingariya/kaku/logger"
	"github.com/piyushsingariya/kaku/models"
	"github.com/piyushsingariya/kaku/types"
	"github.com/piyushsingariya/kaku/typing"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// IsValidSubcommand checks if the passed subcommand is supported by the parent command
func IsValidSubcommand(available []*cobra.Command, sub string) bool {
	for _, s := range available {
		if sub == s.CalledAs() {
			return true
		}
	}
	return false
}

func ArrayContains[T comparable](array []T, value T) bool {
	for _, elem := range array {
		if elem == value {
			return true
		}
	}

	return false
}

func ToJSONSchema(obj interface{}) (string, error) {
	schema, err := jsonschema.Reflect(obj)
	if err != nil {
		return "", err
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func ToYamlSchema(obj interface{}) (string, error) {
	jsonSchema, err := ToJSONSchema(obj)
	if err != nil {
		return "", err
	}

	yamlData, err := yaml.JSONToYAML([]byte(jsonSchema))
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

// Unmarshal serializes and deserializes any from into the object
// return error if occurred
func Unmarshal(from interface{}, object interface{}) error {
	reformatted := reformatInnerMaps(from)
	b, err := json.Marshal(reformatted)
	if err != nil {
		return fmt.Errorf("error marshalling object: %v", err)
	}
	err = json.Unmarshal(b, object)
	if err != nil {
		return fmt.Errorf("error unmarshalling from object: %v", err)
	}

	return nil
}

func IsInstance(val any, typ reflect.Kind) bool {
	return reflect.ValueOf(val).Kind() == typ
}

// reformatInnerMaps converts all map[interface{}]interface{} into map[string]interface{}
// because json.Marshal doesn't support map[interface{}]interface{} (supports only string keys)
// but viper produces map[interface{}]interface{} for inner maps
// return recursively converted all map[interface]interface{} to map[string]interface{}
func reformatInnerMaps(valueI interface{}) interface{} {
	switch value := valueI.(type) {
	case []interface{}:
		for i, subValue := range value {
			value[i] = reformatInnerMaps(subValue)
		}
		return value
	case map[interface{}]interface{}:
		newMap := make(map[string]interface{}, len(value))
		for k, subValue := range value {
			newMap[fmt.Sprint(k)] = reformatInnerMaps(subValue)
		}
		return newMap
	case map[string]interface{}:
		for k, subValue := range value {
			value[k] = reformatInnerMaps(subValue)
		}
		return value
	default:
		return valueI
	}
}

func CheckIfFilesExists(files ...string) error {
	for _, file := range files {
		// Check if the file or directory exists
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist: %s", file, err)
		}

		_, err = os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %s", file, err)
		}
	}

	return nil
}

func ReadFile(file string) interface{} {
	content, err := ReadFileE(file)
	if err != nil {
		logger.Error(err)
		return nil
	}

	return content
}

func ReadFileE(file string) (interface{}, error) {
	if err := CheckIfFilesExists(file); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("file not found : %s", err)
	}

	var content interface{}

	err = yaml.Unmarshal(data, &content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func IsOfType(object interface{}, decidingKey string) (bool, error) {
	objectMap := make(map[string]interface{})
	if err := Unmarshal(object, &objectMap); err != nil {
		return false, err
	}

	if _, found := objectMap[decidingKey]; found {
		return true, nil
	}

	return false, nil
}

func StreamIdentifier(namespace, name string) string {
	return namespace + name
}

func ToKakuSchema(obj interface{}) (string, error) {
	schema, err := jsonschema.Reflect(obj)
	if err != nil {
		return "", err
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func RetryOnFailure(attempts int, sleep *time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}

		logger.Infof("Retrying after %v...", sleep)
		time.Sleep(*sleep)
	}

	return err
}

func IsSubset[T comparable](setArray, subsetArray []T) bool {
	set := make(map[T]bool)
	for _, item := range setArray {
		set[item] = true
	}

	for _, item := range subsetArray {
		if _, found := set[item]; !found {
			return false
		}
	}

	return true
}

func MaxDate(v1, v2 time.Time) time.Time {
	if v1.After(v2) {
		return v1
	}

	return v2
}

func MaximumOnDataType[T any](typ []types.DataType, a, b T) (T, error) {
	switch {
	case ArrayContains(typ, types.TIMESTAMP):
		adate, err := typing.ReformatDate(a)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", a, err)
		}
		bdate, err := typing.ReformatDate(b)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", b, err)
		}

		if MaxDate(adate, bdate) == adate {
			return a, nil
		}

		return b, nil
	default:
		return a, fmt.Errorf("comparison not available for data types %v now", typ)
	}
}

func ReformatRecord(name, namespace string, record map[string]any) models.Record {
	return models.Record{
		Stream:    name,
		Namespace: namespace,
		Data:      record,
	}
}
