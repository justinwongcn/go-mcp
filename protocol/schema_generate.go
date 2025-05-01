package protocol

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// DataType 定义JSON Schema支持的数据类型
// [协议规范] 遵循JSON Schema Draft 07标准
type DataType string

const (
	ObjectT DataType = "object"  // 对象类型
	Number  DataType = "number"  // 数字类型(包含浮点数)
	Integer DataType = "integer" // 整数类型
	String  DataType = "string"  // 字符串类型
	Array   DataType = "array"   // 数组类型
	Null    DataType = "null"    // null类型
	Boolean DataType = "boolean" // 布尔类型
)

// Property 定义JSON Schema的属性结构
// [重要] 用于描述JSON数据的结构和约束条件
type Property struct {
	Type DataType `json:"type"`
	// Description is the description of the schema.
	Description string `json:"description,omitempty"`
	// Items specifies which data type an array contains, if the schema type is Array.
	Items *Property `json:"items,omitempty"`
	// Properties describes the properties of an object, if the schema type is Object.
	Properties map[string]*Property `json:"properties,omitempty"`
	Required   []string             `json:"required,omitempty"`
	Enum       []string             `json:"enum,omitempty"`
}

var schemaCache = pkg.SyncMap[*InputSchema]{}

// generateSchemaFromReqStruct 从请求结构体生成JSON Schema
// [性能提示] 使用缓存机制避免重复生成相同类型的schema
// 输入参数:
//   - v: 可以是结构体实例或指针
//
// 返回值:
//   - *InputSchema: 生成的schema
//   - error: 类型不匹配或枚举值无效时返回错误
func generateSchemaFromReqStruct(v any) (*InputSchema, error) {
	t := reflect.TypeOf(v)
	for t.Kind() != reflect.Struct {
		if t.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("invalid type %v", t)
		}
		t = t.Elem()
	}

	typeUID := getTypeUUID(t)
	if schema, ok := schemaCache.Load(typeUID); ok {
		return schema, nil
	}

	schema := &InputSchema{Type: Object}

	property, err := reflectSchemaByObject(t)
	if err != nil {
		return nil, err
	}

	schema.Properties = property.Properties
	schema.Required = property.Required

	schemaCache.Store(typeUID, schema)
	return schema, nil
}

func getTypeUUID(t reflect.Type) string {
	if t.PkgPath() != "" && t.Name() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	// fallback for unnamed types (like anonymous struct)
	return t.String()
}

// reflectSchemaByObject 通过反射从结构体类型生成Property
// [算法说明] 递归处理结构体字段，生成完整的属性定义
// 处理逻辑:
// 1. 解析json标签确定字段名和是否必填
// 2. 根据字段类型生成对应的Property
// 3. 处理description和enum标签
// 4. 校验枚举值与字段类型的兼容性
func reflectSchemaByObject(t reflect.Type) (*Property, error) {
	var (
		properties     = make(map[string]*Property)
		requiredFields = make([]string, 0)
		enumValues     = make([]string, 0)
	)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		required := true
		if jsonTag == "" {
			jsonTag = field.Name
		}
		if strings.HasSuffix(jsonTag, ",omitempty") {
			jsonTag = strings.TrimSuffix(jsonTag, ",omitempty")
			required = false
		}

		item, err := reflectSchemaByType(field.Type)
		if err != nil {
			return nil, err
		}

		if description := field.Tag.Get("description"); description != "" {
			item.Description = description
		}
		properties[jsonTag] = item

		if s := field.Tag.Get("required"); s != "" {
			required, err = strconv.ParseBool(s)
			if err != nil {
				return nil, fmt.Errorf("invalid required field %v: %v", jsonTag, err)
			}
		}
		if required {
			requiredFields = append(requiredFields, jsonTag)
		}

		if v := field.Tag.Get("enum"); v != "" {
			enumValues = strings.Split(v, ",")
			for i := range enumValues {
				enumValues[i] = strings.TrimSpace(enumValues[i])
			}

			// Check if enum values are consistent with the field type
			for _, value := range enumValues {
				switch field.Type.Kind() {
				case reflect.String:
					// No additional processing required for string type
				case reflect.Int, reflect.Int64:
					if _, err := strconv.Atoi(value); err != nil {
						return nil, fmt.Errorf("enum value %q is not compatible with type %v", value, field.Type)
					}
				case reflect.Float64:
					if _, err := strconv.ParseFloat(value, 64); err != nil {
						return nil, fmt.Errorf("enum value %q is not compatible with type %v", value, field.Type)
					}
				default:
					return nil, fmt.Errorf("unsupported type %v for enum validation", field.Type)
				}
			}
			item.Enum = enumValues
		}
	}

	property := &Property{
		Type:       ObjectT,
		Properties: properties,
		Required:   requiredFields,
		Enum:       enumValues,
	}
	return property, nil
}

// reflectSchemaByType 通过反射从类型生成Property
// [注意] 处理所有支持的基本类型和复合类型
// 支持的类型包括:
// - 基本类型: string, number, integer, boolean
// - 复合类型: array, object
// - 特殊类型: null
// 不支持的复杂类型会返回错误
func reflectSchemaByType(t reflect.Type) (*Property, error) {
	s := &Property{}

	switch t.Kind() {
	case reflect.String:
		s.Type = String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s.Type = Integer
	case reflect.Float32, reflect.Float64:
		s.Type = Number
	case reflect.Bool:
		s.Type = Boolean
	case reflect.Slice, reflect.Array:
		s.Type = Array
		items, err := reflectSchemaByType(t.Elem())
		if err != nil {
			return nil, err
		}
		s.Items = items
	case reflect.Struct:
		object, err := reflectSchemaByObject(t)
		if err != nil {
			return nil, err
		}
		object.Type = ObjectT
		s = object
	case reflect.Ptr:
		p, err := reflectSchemaByType(t.Elem())
		if err != nil {
			return nil, err
		}
		s = p
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.UnsafePointer:
		return nil, fmt.Errorf("unsupported type: %s", t.Kind().String())
	default:
	}
	return s, nil
}
