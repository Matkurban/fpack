---
title: "JSON Schema"
description: "为 fpack.yaml 提供编辑器补全、悬停文档和校验。"
---

fpack 为 `fpack.yaml` 发布了 JSON Schema，它与驱动校验、`fpack init` 和这些文档的是同一份键注册表生成的。有了它，编辑器可以补全键、显示每个键的文档（中英文），并在你输入时标出错误。

## 自动方式

`fpack init` 会把下面这行写在 `fpack.yaml` 的第一行：

```text
# yaml-language-server: $schema=https://raw.githubusercontent.com/Matkurban/fpack/main/schema/fpack.schema.json
```

- **VS Code**：安装 [YAML 扩展](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)，它会读取这一行。
- **JetBrains IDE**（IntelliJ、Android Studio）：YAML 支持同样识别这一行（也可以在 *Json Schema* 设置里手动映射文件）。

## 离线副本

```bash
fpack schema -o .vscode/fpack.schema.json
```

让编辑器指向本地文件，例如在 VS Code 的 `settings.json` 中：

```json
{
  "yaml.schemas": { ".vscode/fpack.schema.json": "fpack.yaml" }
}
```

不带 `-o` 的 `fpack schema` 输出到 stdout，可以管道到任何地方。

## 它校验什么

- 键名（不允许未知键）、嵌套的段、类型、允许的值和数值范围；
- 悬停文本，包含说明、默认值、示例以及对应的 `FPACK_*` 变量或参数。

fpack 每次运行时也会用同样的规则校验文件，并加上 schema 表达不了的检查（例如设置了签名密码却没有密钥文件）。见[键参考](/configuration)。
