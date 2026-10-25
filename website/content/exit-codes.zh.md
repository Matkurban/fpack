---
title: "退出码与 JSON 输出"
description: "每个退出码的含义，以及供脚本和 CI 使用的机器可读 JSON 结果。"
---

脚本和 CI 任务应先看退出码，再看 JSON 结果了解细节。

## 退出码

<!-- BEGIN GENERATED EXIT -->
| 退出码 | 含义 |
| --- | --- |
| 0 | 成功（`--dry-run`：计划成功生成） |
| 1 | 至少一个目标在构建/打包时失败，或钩子失败；`fpack doctor`：发现问题；`fpack notarize`：有提交被拒绝或查询失败 |
| 2 | 用法或配置错误（未知参数或目标、未指定目标且没有 `build.targets`、fpack.yaml 有误、找不到 Flutter 项目） |
| 3 | 前置条件不满足 / 产物已存在 / 没有可构建的目标 / 找不到 Flutter SDK；`fpack notarize`：找不到公证记录 |
| 130 | 被 Ctrl-C 中断（公证等待被中断时，Apple 端仍会继续处理） |
<!-- END GENERATED EXIT -->

据此做出反应：`3` 表示缺少前置条件，什么都没构建（修好机器，不要盲目重试）；`2` 表示你的命令或配置有误；`1` 表示构建或检查已经运行但失败了。

```bash
fpack build apk aab || echo "fpack 退出码 $?"
```

<a id="json-zh"></a>

## JSON 输出

`--json` 向 stdout 恰好输出一个 JSON 文档；所有人类可读的输出走 stderr，所以可以放心重定向：

```bash
fpack build apk aab --json > result.json
fpack doctor --json
fpack list --json
```

一次成功的 Web 构建大致如下（已缩短）：

```json
{
  "fpackVersion": "1.1.17",
  "project": "my_app",
  "version": "1.0.0",
  "buildNumber": "1",
  "flutter": "3.47.7",
  "host": "linux/amd64",
  "mode": "release",
  "outputDir": "/work/my_app/dist/1.0.0+1",
  "checksumsFile": "/work/my_app/dist/1.0.0+1/SHA256SUMS",
  "logDir": "/work/my_app/build/fpack/logs",
  "targets": [
    {
      "target": "web",
      "platform": "web",
      "status": "success",
      "artifacts": [
        {
          "path": "/work/my_app/dist/1.0.0+1/my_app-1.0.0+1-web.zip",
          "file": "my_app-1.0.0+1-web.zip",
          "kind": "Web (zip)",
          "size": 14431167,
          "sha256": "7f78faa3…"
        }
      ],
      "durationMs": 22802
    }
  ],
  "exitCode": 0,
  "durationMs": 22813,
  "dryRun": false,
  "success": true
}
```

### `fpack build --json` 的字段

| 字段 | 含义 |
| --- | --- |
| `fpackVersion`、`project`、`projectRoot`、`version`、`buildNumber`、`flutter`、`flutterRoot`、`host`、`mode`、`flavor` | 构建了什么、用的什么工具 |
| `outputDir`、`checksumsFile`、`logDir` | 产物、校验和与日志的位置 |
| `targets[]` | 每个目标一项 |
| `targets[].status` | `success`、`failed`、`skipped` 或 `planned`（预演） |
| `targets[].reason`、`fix` | 目标为何被跳过，以及如何让它可以构建 |
| `targets[].error`、`hint`、`log`、`errorExcerpt[]` | 失败详情：错误信息、建议的修复、完整日志路径、关键行 |
| `targets[].warnings[]`、`notes[]`、`commands[]` | 警告、说明，以及已运行（或将运行）的命令 |
| `targets[].artifacts[]` | `path`、`file`、`kind`、`arch`、`size`、`sha256`（以及 `sha512`） |
| `targets[].artifacts[].notarization` | 已公证的 macOS 文件：`state`（`submitted`、`accepted`、`stapled`、`invalid`）、`id`、`status`、`record`（`NOTARIZATION.md` 路径） |
| `targets[].durationMs`、`durationMs` | 耗时 |
| `hookError` | 失败钩子的错误 |
| `hooks[]` | 预演中将会执行的钩子命令 |
| `conflicts[]` | 真正运行时会因此停止的已有产物（用 `--force` 替换） |
| `exitCode`、`success`、`dryRun`、`interrupted` | 结果 |

### 示例

```bash
# 上传所有产物
jq -r '.targets[].artifacts[].path' result.json

# 所有目标都成功了吗？
jq -e '.success' result.json

# 列出被跳过的目标及原因
jq -r '.targets[] | select(.status=="skipped") | "\(.target): \(.reason)"' result.json
```

## 下一步

[CI 示例](/ci)把这些用起来；[故障排查](/troubleshooting)说明每种失败该怎么办。
