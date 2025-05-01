package pkg

import (
	"encoding/json"
	"fmt"
)

// var sonicAPI = sonic.Config{UseInt64: true}.Froze() // Effectively prevents integer overflow

// JSONUnmarshal 解析JSON数据并返回格式化错误
// data: JSON字节数据
// v: 目标解析对象
// 返回: 错误信息
func JSONUnmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: data=%s, error: %+v", ErrJSONUnmarshal, data, err)
	}
	return nil
}
