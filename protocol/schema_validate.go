package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
)

// VerifyAndUnmarshal 验证JSON数据并反序列化到目标结构体
// [重要] 核心验证逻辑，确保数据符合schema定义
// 处理流程:
// 1. 检查空内容
// 2. 验证目标类型是否为结构体或指针
// 3. 从缓存获取schema进行验证
// 4. 调用底层验证和反序列化函数
func VerifyAndUnmarshal(content json.RawMessage, v any) error {
	if len(content) == 0 {
		return fmt.Errorf("request arguments is empty")
	}

	t := reflect.TypeOf(v)
	for t.Kind() != reflect.Struct {
		if t.Kind() != reflect.Ptr {
			return fmt.Errorf("invalid type %v, plz use func `pkg.JSONUnmarshal` instead", t)
		}
		t = t.Elem()
	}

	typeUID := getTypeUUID(t)
	schema, ok := schemaCache.Load(typeUID)
	if !ok {
		return fmt.Errorf("schema has not been generated，unable to verify: plz use func `pkg.JSONUnmarshal` instead")
	}

	return verifySchemaAndUnmarshal(Property{
		Type:       ObjectT,
		Properties: schema.Properties,
		Required:   schema.Required,
	}, content, v)
}

// verifySchemaAndUnmarshal 执行实际的schema验证和反序列化
// [性能提示] 先验证后反序列化，避免无效数据的处理开销
// 输入参数:
//   - schema: 验证用的Property schema
//   - content: 原始JSON数据
//   - v: 目标反序列化结构体
func verifySchemaAndUnmarshal(schema Property, content []byte, v any) error {
	var data any
	err := pkg.JSONUnmarshal(content, &data)
	if err != nil {
		return err
	}
	if !validate(schema, data) {
		return errors.New("data validation failed against the provided schema")
	}
	return pkg.JSONUnmarshal(content, &v)
}

// validate 根据schema验证数据
// [算法说明] 递归验证所有数据类型和嵌套结构
// 支持验证的类型包括:
// - ObjectT: 对象类型
// - Array: 数组类型
// - String: 字符串类型
// - Number: 数字类型
// - Boolean: 布尔类型
// - Integer: 整数类型
// - Null: null类型
func validate(schema Property, data any) bool {
	switch schema.Type {
	case ObjectT:
		return validateObject(schema, data)
	case Array:
		return validateArray(schema, data)
	case String:
		str, ok := data.(string)
		if ok {
			return validateEnumProperty[string](str, schema.Enum, func(value string, enumValue string) bool {
				return value == enumValue
			})
		}
		return false
	case Number: // float64 and int
		if num, ok := data.(float64); ok {
			return validateEnumProperty[float64](num, schema.Enum, func(value float64, enumValue string) bool {
				if enumNum, err := strconv.ParseFloat(enumValue, 64); err == nil && value == enumNum {
					return true
				}
				return false
			})
		}
		if num, ok := data.(int); ok {
			return validateEnumProperty[int](num, schema.Enum, func(value int, enumValue string) bool {
				if enumNum, err := strconv.Atoi(enumValue); err == nil && value == enumNum {
					return true
				}
				return false
			})
		}
		return false
	case Boolean:
		_, ok := data.(bool)
		return ok
	case Integer:
		// Golang unmarshals all numbers as float64, so we need to check if the float64 is an integer
		if num, ok := data.(float64); ok {
			if num == float64(int64(num)) {
				return validateEnumProperty[float64](num, schema.Enum, func(value float64, enumValue string) bool {
					if enumNum, err := strconv.ParseFloat(enumValue, 64); err == nil && value == enumNum {
						return true
					}
					return false
				})
			}
			return false
		}

		if num, ok := data.(int); ok {
			return validateEnumProperty[int](num, schema.Enum, func(value int, enumValue string) bool {
				if enumNum, err := strconv.Atoi(enumValue); err == nil && value == enumNum {
					return true
				}
				return false
			})
		}

		if num, ok := data.(int64); ok {
			return validateEnumProperty[int64](num, schema.Enum, func(value int64, enumValue string) bool {
				if enumNum, err := strconv.Atoi(enumValue); err == nil && value == int64(enumNum) {
					return true
				}
				return false
			})
		}
		return false
	case Null:
		return data == nil
	default:
		return false
	}
}

// validateObject 验证对象类型数据
// [注意] 处理必填字段检查和属性递归验证
func validateObject(schema Property, data any) bool {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return false
	}
	for _, field := range schema.Required {
		if _, exists := dataMap[field]; !exists {
			return false
		}
	}
	for key, valueSchema := range schema.Properties {
		value, exists := dataMap[key]
		if exists && !validate(*valueSchema, value) {
			return false
		}
	}
	return true
}

// validateArray 验证数组类型数据
// [注意] 递归验证数组每个元素
func validateArray(schema Property, data any) bool {
	dataArray, ok := data.([]any)
	if !ok {
		return false
	}
	for _, item := range dataArray {
		if !validate(*schema.Items, item) {
			return false
		}
	}
	return true
}

// validateEnumProperty 验证枚举值
// [设计决策] 使用泛型支持多种类型的枚举验证
func validateEnumProperty[T any](data T, enum []string, compareFunc func(T, string) bool) bool {
	for _, enumValue := range enum {
		if compareFunc(data, enumValue) {
			return true
		}
	}
	return len(enum) == 0
}
