# ClipTool - 剪贴板图片转GIF工具

自动监听剪贴板，将多张图片合成为GIF动画。

## 功能特性

- **自动监听**: 实时检测剪贴板图片变化
- **多图合成**: 支持 2 张及以上图片自动生成 GIF
- **格式支持**: BMP (1/4/8/16/24/32位) / PNG / JPG
- **动画预览**: 左侧显示所有帧，右侧循环播放

## 快速开始

### 运行程序
```bash
.\cliptool.exe
```
或直接双击运行

### 使用流程
1. 启动程序，开始监听剪贴板
2. 复制 2 张或更多图片
3. 程序自动生成 GIF 并写入剪贴板
4. 直接粘贴使用

## 技术实现

### 核心依赖
- `github.com/jsummers/gobmp` - 完整 BMP 格式支持（包括 4 位工业 BMP）
- `github.com/nfnt/resize` - 高性能图片缩放
- `golang.org/x/sys/windows` - Windows 剪贴板 API

### 关键优化
- Bilinear 插值快速缩放
- 无抖动量化（draw.Src）提升 GIF 编码性能
- 256 色自适应调色板（216 色 RGB + 40 级灰度）

## 开发

### 编译
```bash
.\build.bat
```
或手动编译：
```bash
go build -ldflags="-s -w" -o cliptool.exe main.go
```

### 配置
修改 `main.go` 中的常量：
```go
const (
    pollInterval  = 300    // 剪贴板轮询间隔(ms)
    gifDelay      = 50     // GIF帧延迟(10ms单位, 50=500ms)
    minImages     = 2      // 最少图片数
    margin        = 10     // 帧间距(px)
)
```

## 项目结构

```
cliptool-go/
├── main.go      # 主程序源码
├── build.bat    # 编译脚本
├── go.mod       # Go 模块配置
└── README.md    # 项目文档
```

## License

MIT
