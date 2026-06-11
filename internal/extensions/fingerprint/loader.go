package fingerprint

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"cliptool/internal/applog"
)

type Loader struct{}

type rawDataProfile struct {
	sensor        string
	name          string
	width, height int
	dtype         string
	bits          int
}

var rawDataProfiles = map[int64][]rawDataProfile{
	1560:   {{sensor: "0103", name: "RAW", width: 13, height: 60, dtype: "uint16", bits: 12}},
	4800:   {{sensor: "0107", name: "RAW", width: 24, height: 100, dtype: "uint16", bits: 12}},
	6240:   {{sensor: "0103", name: "RAW", width: 26, height: 120, dtype: "uint16", bits: 12}, {sensor: "0106", name: "RAW", width: 20, height: 156, dtype: "uint16", bits: 12}},
	8320:   {{sensor: "0104", name: "RAW", width: 26, height: 160, dtype: "uint16", bits: 12}},
	9600:   {{sensor: "0107", name: "BIN", width: 24, height: 100, dtype: "uint32", bits: 16}},
	10240:  {{sensor: "0101", name: "RAW", width: 32, height: 160, dtype: "uint16", bits: 12}},
	12480:  {{sensor: "0103", name: "BIN", width: 26, height: 120, dtype: "uint32", bits: 16}, {sensor: "0106", name: "BIN", width: 20, height: 156, dtype: "uint32", bits: 16}},
	16640:  {{sensor: "0104", name: "BIN", width: 26, height: 160, dtype: "uint32", bits: 16}},
	20480:  {{sensor: "0101", name: "BIN", width: 32, height: 160, dtype: "uint32", bits: 16}},
	39192:  {{sensor: "0201", name: "RAW", width: 138, height: 142, dtype: "uint16", bits: 10}},
	43808:  {{sensor: "0307", name: "RAW", width: 148, height: 148, dtype: "uint16", bits: 12}},
	51200:  {{sensor: "0301", name: "RAW", width: 160, height: 160, dtype: "uint16", bits: 12}, {sensor: "0302", name: "RAW", width: 160, height: 160, dtype: "uint16", bits: 10}, {sensor: "0401", name: "RAW", width: 160, height: 160, dtype: "uint16", bits: 12}},
	88200:  {{sensor: "0305", name: "RAW", width: 210, height: 210, dtype: "uint16", bits: 12}},
	102400: {{sensor: "0307", name: "BIN", width: 160, height: 160, dtype: "uint32", bits: 16}, {sensor: "0301", name: "BIN", width: 160, height: 160, dtype: "uint32", bits: 16}, {sensor: "0302", name: "BIN", width: 160, height: 160, dtype: "uint32", bits: 16}, {sensor: "0401", name: "BIN", width: 160, height: 160, dtype: "uint32", bits: 16}},
	176400: {{sensor: "0305", name: "BIN", width: 210, height: 210, dtype: "uint32", bits: 16}},
}

func NewLoader() Loader {
	return Loader{}
}

func (Loader) Name() string {
	return "fingerprint-raw-bin"
}

func (Loader) IsSupported(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	_, ok := selectRawDataProfile(filePath, info.Size())
	return ok
}

func (Loader) Load(filePath string) (image.Image, string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		applog.Errorf("指纹扩展文件状态检查失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}

	profile, ok := selectRawDataProfile(filePath, info.Size())
	if !ok {
		return nil, "", fmt.Errorf("unknown fingerprint raw/bin size: %d", info.Size())
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		applog.Errorf("读取指纹扩展数据失败: path=%q err=%v", filePath, err)
		return nil, "", err
	}

	img := image.NewGray(image.Rect(0, 0, profile.width, profile.height))
	pixelCount := profile.width * profile.height
	expectedBytes := pixelCount * rawDataBytesPerPixel(profile)
	if len(data) < expectedBytes {
		return nil, "", fmt.Errorf("fingerprint raw data too short: %d, expected %d", len(data), expectedBytes)
	}

	for i := 0; i < pixelCount; i++ {
		var rawValue uint32
		if profile.dtype == "uint16" {
			rawValue = uint32(binary.LittleEndian.Uint16(data[i*2:]))
		} else {
			rawValue = binary.LittleEndian.Uint32(data[i*4:])
		}
		img.Pix[i] = uint8(rawValue >> (profile.bits - 8))
	}

	format := fmt.Sprintf("%s/%dx%d", profile.name, profile.width, profile.height)
	applog.Debugf("指纹扩展读取成功: path=%q format=%q bytes=%d", filePath, format, len(data))
	return img, format, nil
}

func selectRawDataProfile(filePath string, size int64) (rawDataProfile, bool) {
	if profiles, ok := rawDataProfiles[size]; ok && len(profiles) > 0 {
		return selectRawDataProfileCandidate(filePath, size, profiles), true
	}

	return rawDataProfile{}, false
}

func selectRawDataProfileCandidate(filePath string, size int64, profiles []rawDataProfile) rawDataProfile {
	if len(profiles) == 1 {
		return profiles[0]
	}

	normalizedPath := strings.ToLower(filePath)
	for _, profile := range profiles {
		if profile.sensor != "" && strings.Contains(normalizedPath, profile.sensor) {
			return profile
		}
	}

	baseName := strings.ToLower(filepath.Base(filePath))
	for _, profile := range profiles {
		if profile.name == "RAW" && strings.HasPrefix(baseName, "raw_") {
			return profile
		}
		if profile.name == "BIN" && strings.HasPrefix(baseName, "bin_") {
			return profile
		}
	}

	applog.Warnf("指纹扩展尺寸存在多个候选，未从路径识别 sensor，使用默认候选: path=%q size=%d sensor=%q width=%d height=%d",
		filePath, size, profiles[0].sensor, profiles[0].width, profiles[0].height)
	return profiles[0]
}

func rawDataBytesPerPixel(profile rawDataProfile) int {
	if profile.dtype == "uint16" {
		return 2
	}
	return 4
}
