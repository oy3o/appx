package appx

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
)

// printServiceListening 打印服务启动横幅
func printServiceListening(logger *zerolog.Logger, name, protocol, addr string) {
	if logger == nil {
		return
	}

	pid := os.Getpid()

	// 尝试获取更友好的 IP 显示 (例如将 [::] 替换为 localhost 或具体 IP，视情况而定)
	// 这里保持真实 Listener 地址以确保准确性
	logger.Info().
		Str("service", name).
		Str("protocol", protocol).
		Str("address", addr).
		Int("pid", pid).
		Msg("Service listening...")
}

// printConfigSnapshot 打印脱敏后的配置快照
func printConfigSnapshot(logger *zerolog.Logger, cfg any) {
	if cfg == nil || logger == nil {
		return
	}

	masked := maskSensitiveData(cfg)

	// 格式化为 JSON
	b, err := sonic.MarshalIndent(masked, "", "  ")
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to marshal config snapshot")
		return
	}

	logger.Info().RawJSON("config_snapshot", b).Msg("Effective Configuration")
}

// maskSensitiveData 递归遍历结构体或 Map，对敏感字段进行脱敏
func maskSensitiveData(v any) any {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	// 解引用指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		out := make(map[string]any)
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			// 跳过未导出字段
			if !field.IsExported() {
				continue
			}

			fieldName := field.Name
			// 优先使用 mapstructure > json > yaml 标签作为 Key
			if tag := field.Tag.Get("mapstructure"); tag != "" && tag != "-" {
				fieldName = strings.Split(tag, ",")[0]
			} else if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				fieldName = strings.Split(tag, ",")[0]
			}

			fieldVal := val.Field(i).Interface()

			// 检查是否是敏感字段
			if isSensitive(fieldName) {
				out[fieldName] = "******"
			} else {
				out[fieldName] = maskSensitiveData(fieldVal)
			}
		}
		return out

	case reflect.Map:
		out := make(map[string]any)
		for _, k := range val.MapKeys() {
			keyStr := fmt.Sprint(k.Interface())
			mapVal := val.MapIndex(k).Interface()

			if isSensitive(keyStr) {
				out[keyStr] = "******"
			} else {
				out[keyStr] = maskSensitiveData(mapVal)
			}
		}
		return out

	case reflect.Slice, reflect.Array:
		out := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			out[i] = maskSensitiveData(val.Index(i).Interface())
		}
		return out

	default:
		return v
	}
}

// isSensitive 判断字段名是否包含敏感词
func isSensitive(name string) bool {
	name = strings.ToLower(name)
	keywords := []string{"password", "secret", "token", "key", "auth", "credential", "pwd"}
	for _, kw := range keywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}
