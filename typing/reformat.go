package typing

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/piyushsingariya/kaku/types"
)

var DateTimeFormats = []string{
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05 -07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.000000",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05+0000",
	"2020-08-17T05:50:22.895Z",
}

func getFirstNotNullType(datatypes []types.DataType) types.DataType {
	for _, datatype := range datatypes {
		if datatype != types.NULL {
			return datatype
		}
	}

	return types.NULL
}

func ReformatValueOnDataTypes(datatypes []types.DataType, v any) (any, error) {
	return ReformatValue(getFirstNotNullType(datatypes), v)
}

func ReformatValue(dataType types.DataType, v any) (any, error) {
	switch dataType {
	case types.NULL:
		return nil, nil
	case types.BOOL:
		switch booleanValue := v.(type) {
		case bool:
			return booleanValue, nil
		case string:
			switch booleanValue {
			case "1", "t", "T", "true", "TRUE", "True", "YES", "Yes", "yes":
				return true, nil
			case "0", "f", "F", "false", "FALSE", "False", "NO", "No", "no":
				return false, nil
			}
		case int, int16, int32, int64, int8:
			switch booleanValue {
			case 1:
				return true, nil
			case 0:
				return true, nil
			default:
				return nil, fmt.Errorf("found to be boolean, but value is not boolean : %v", v)
			}
		default:
			return nil, fmt.Errorf("found to be boolean, but value is not boolean : %v", v)
		}

		return nil, fmt.Errorf("found to be boolean, but value is not boolean : %v", v)
	case types.INT64:
		return ReformatInt64(v)
	case types.TIMESTAMP:
		return ReformatDate(v)
	case types.STRING:
		return fmt.Sprintf("%v", v), nil
	case types.FLOAT64:
		return ReformatFloat64(v)
	case types.ARRAY:
		if value, isArray := v.([]any); isArray {
			return value, nil
		}

		// make it an array
		return []any{v}, nil
	default:
		return v, nil
	}
}

// reformat date
func ReformatDate(v interface{}) (time.Time, error) {
	parsed, err := func() (time.Time, error) {
		switch v := v.(type) {
		// we assume int64 is in seconds and don't currently scale to the precision
		case int64:
			return time.Unix(v, 0), nil
		case *int64:
			switch {
			case v != nil:
				return time.Unix(*v, 0), nil
			default:
				return time.Time{}, fmt.Errorf("null time passed")
			}
		case time.Time:
			return v, nil
		case *time.Time:
			switch {
			case v != nil:
				return *v, nil
			default:
				return time.Time{}, fmt.Errorf("null time passed")
			}
		case sql.NullTime:
			switch v.Valid {
			case true:
				return v.Time, nil
			default:
				return time.Time{}, fmt.Errorf("invalid null time")
			}
		case *sql.NullTime:
			switch v.Valid {
			case true:
				return v.Time, nil
			default:
				return time.Time{}, fmt.Errorf("invalid null time")
			}
		case nil:
			return time.Time{}, nil
		case string:
			return parseCHDateTime(v)
		case *string:
			if v == nil || *v == "" {
				return time.Time{}, fmt.Errorf("empty string passed")
			} else {
				return parseCHDateTime(*v)
			}
		}
		return time.Time{}, fmt.Errorf("unknown type passed: unable to parse into time")
	}()
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}

func parseCHDateTime(value string) (time.Time, error) {
	var tv time.Time
	var err error
	for _, layout := range DateTimeFormats {
		tv, err = time.Parse(layout, value)
		if err == nil {
			return time.Date(
				tv.Year(), tv.Month(), tv.Day(), tv.Hour(), tv.Minute(), tv.Second(), tv.Nanosecond(), tv.Location(),
			), nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse datetime from available formats: %v : %s", DateTimeFormats, err)
}

func ReformatInt64(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	}

	return int64(0), fmt.Errorf("failed to change %v (type:%T) to int64", v, v)
}

func ReformatFloat64(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return float64(0), fmt.Errorf("failed to change string %v to float64: %w", v, err)
		}
		return f, nil
	}

	return float64(0), fmt.Errorf("failed to change %v (type:%T) to float64", v, v)
}
