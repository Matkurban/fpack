---
title: "钩子"
description: "在构建或打包前后运行你自己的脚本：上传、检查、通知。"
---

钩子是 fpack 在固定时刻运行的 shell 命令。可以用它在构建前生成代码、检查产物大小、上传到商店或服务器，或者发送通知。

```yaml
hooks:
  pre_build:
    - dart run build_runner build --delete-conflicting-outputs
  pre_package:
    apk: ['./scripts/check_size.sh']
  post_package:
    dmg: ['./scripts/upload.sh "$FPACK_ARTIFACT"']
  post_build:
    - ./scripts/notify.sh
```

## 各钩子何时运行

| 钩子 | 时机 | 说明 |
| --- | --- | --- |
| `hooks.pre_build` | 第一次 flutter 构建之前，运行一次 | 失败会中止整个运行 |
| `hooks.pre_package.<目标>` | 某个目标的打包步骤之前 | 设置 `FPACK_TARGET` |
| `hooks.post_package.<目标>` | 某个目标的产物生成之后 | 设置 `FPACK_ARTIFACT`（第一个文件）和 `FPACK_ARTIFACTS` |
| `hooks.post_build` | 最后运行一次 | 设置 `FPACK_ARTIFACTS`（每行一个）和 `FPACK_SUCCESS`（`1` / `0`）。构建失败后也会运行，但所有目标都在检查阶段就失败时（缺少工具、未知 flavor）不会运行 |

钩子失败意味着退出码 1。

## 环境

钩子在项目根目录通过 `sh -c` 运行（Windows 是 `cmd /S /C`，所以带引号的路径照常使用），并带有这些变量：

| 变量 | 值 |
| --- | --- |
| `FPACK_PROJECT_ROOT` | 项目目录 |
| `FPACK_OUTPUT_DIR` | 产物目录 |
| `FPACK_VERSION`、`FPACK_BUILD_NUMBER` | 版本名和构建号 |
| `FPACK_MODE`、`FPACK_FLAVOR` | 构建模式和 flavor |
| `FPACK_TARGET` | 目标（`pre_package` / `post_package`） |
| `FPACK_ARTIFACT`、`FPACK_ARTIFACTS` | 产物（`post_package`、`post_build`） |
| `FPACK_SUCCESS` | `1` 或 `0`（`post_build`） |

仅在钩子运行期间才存在的变量，写成 `$VAR`（Windows 的 `cmd` 里是 `%VAR%`）。`${VAR}` 在更早的时候，也就是加载 `fpack.yaml` 时展开（[密钥与环境变量](/guides/secrets)）。

## 示例

打包后上传已签名的 IPA：

```yaml
hooks:
  post_package:
    ipa: ['xcrun altool --upload-app -f "$FPACK_ARTIFACT" -t ios --apiKey "$ASC_KEY_ID" --apiIssuer "$ASC_ISSUER"']
```

在最后一个目标之后，一次性把所有文件复制到发布服务器：

```yaml
hooks:
  post_build:
    - 'test "$FPACK_SUCCESS" = 1 && rsync -a "$FPACK_OUTPUT_DIR/" releases@example.com:/srv/app/'
```

APK 太大时让构建失败：

```yaml
hooks:
  pre_package: { apk: ['./scripts/check_apk_size.sh'] }
```

## 下一步

[多目标一条命令](/guides/many-targets)。
