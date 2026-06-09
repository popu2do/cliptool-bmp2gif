# ClipTool 文档索引

这里放的是 `cliptool-bmp2gif` 的使用说明和交互规格，面向工具使用者和维护者。

## 快速入口

- [使用指南](./cliptool-bmp2gif-user-guide.md) - 先看这个，了解怎么用、怎么出 GIF、常见问题怎么处理。
- [GUI 交互规格](./spec/cliptool-bmp2gif-gui-spec.md) - 保留的交互需求原文，偏设计和约束。

## 适用范围

- 只面向 Windows。
- 主要工作流是“资源管理器复制图片 -> 工具自动收集 -> 调整顺序 -> 生成 GIF -> 粘贴到目标软件”。
- 当前工具不是通用图片编辑器，也不负责保存项目文件或历史记录。
