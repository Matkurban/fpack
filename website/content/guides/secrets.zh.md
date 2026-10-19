---
title: "密钥与环境变量"
description: "哪个设置优先、${VAR} 如何工作、FPACK_* 变量，以及如何让密码不进文件和日志。"
---

签名需要密码和密钥。fpack 的设计目标是：它们永远不必写进 `fpack.yaml`、命令行或日志。

## 哪个设置优先

对每个设置，第一个提供了值的来源生效：

1. 命令行参数（`--flavor prod`）
2. `FPACK_*` 环境变量（`FPACK_FLAVOR=prod`）
3. `fpack.yaml`（`build.flavor: prod`）
4. 默认值

[参考](/configuration)里每个键都列出了它的变量和参数；[环境变量页](/environment)列出全部变量。

## `fpack.yaml` 里的 `${VAR}`

`fpack.yaml` 里的任何值都可以在加载文件时读取环境变量：

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${KEYSTORE_PASSWORD}          # 未设置则报错
    key_alias: ${KEY_ALIAS:-upload}               # 未设置时使用默认值
output:
  dir: ${DIST_DIR:-dist}
```

任何机密或与机器相关的值都用它。文件可以提交，值放在你的 shell、密码管理器或 CI 密钥里。

::: note
`${VAR}` 是在**加载配置时**展开的。在[钩子](/guides/hooks)里，仅在钩子运行期间才存在的变量要写成 `$VAR`（Windows 是 `%VAR%`）。
:::

## 专用变量

签名和公证的设置有各自的变量，所以 CI 任务无需修改 `fpack.yaml`：

| 范围 | 变量 |
| --- | --- |
| Android | `FPACK_ANDROID_KEYSTORE`、`FPACK_ANDROID_KEYSTORE_BASE64`、`FPACK_ANDROID_KEYSTORE_PASSWORD`、`FPACK_ANDROID_KEY_ALIAS`、`FPACK_ANDROID_KEY_PASSWORD` |
| iOS | `FPACK_IOS_EXPORT_METHOD`、`FPACK_IOS_EXPORT_OPTIONS_PLIST`、`FPACK_IOS_CODESIGN` |
| macOS 签名 | `FPACK_MACOS_SIGN`、`FPACK_MACOS_SIGN_IDENTITY`、`FPACK_MACOS_INSTALLER_IDENTITY`、`FPACK_MACOS_PROVISIONING_PROFILE`、`FPACK_MACOS_PROFILE_CHECK`、`FPACK_MACOS_SMOKE_TEST` |
| macOS 公证 | `FPACK_MACOS_NOTARIZE`、`FPACK_MACOS_NOTARY_PROFILE`、`FPACK_NOTARIZE_WAIT`、`FPACK_NOTARY_APPLE_ID`、`FPACK_NOTARY_TEAM_ID`、`FPACK_NOTARY_PASSWORD`、`FPACK_NOTARY_API_KEY`、`FPACK_NOTARY_API_KEY_ID`、`FPACK_NOTARY_API_ISSUER` |
| Windows | `FPACK_WINDOWS_CERTIFICATE`、`FPACK_WINDOWS_CERTIFICATE_PASSWORD`、`FPACK_WINDOWS_CERT_THUMBPRINT`、`FPACK_WINDOWS_CERT_SUBJECT`、`FPACK_WINDOWS_SIGN_DLIB`、`FPACK_WINDOWS_SIGN_DLIB_METADATA` |

## fpack 如何保护它们

- 密码会在日志、`--dry-run` 输出（显示为 `***`）、`NOTARIZATION.md` 和 `--json` 里被隐藏。
- 密码绝不会出现在命令行上；它们通过环境变量或临时文件传递。
- fpack 写出的密钥文件（例如从 `FPACK_ANDROID_KEYSTORE_BASE64` 解码出的密钥库）放在 `build/fpack/secrets/`，权限 0600，**构建后删除**。
- 它在漫长的构建*之前*就检查凭证：错误的密钥库密码或别名、找不到的签名证书。

## 在 CI 中

1. 把密钥存进 CI 的密钥库。
2. 在任务的 `env` 里把它们映射到 `FPACK_*` 变量。
3. 对文件（密钥库、`.p12`、API 密钥 `.p8`），以 base64 存储，在运行机器上解码。

```yaml
- run: fpack build apk aab
  env:
    FPACK_ANDROID_KEYSTORE_BASE64: ${{ secrets.KEYSTORE_BASE64 }}
    FPACK_ANDROID_KEYSTORE_PASSWORD: ${{ secrets.KEYSTORE_PASSWORD }}
    FPACK_ANDROID_KEY_ALIAS: upload
```

各平台的完整工作流：[CI 示例](/ci)。各平台细节：[Android](/platforms/android#在-ci-中)、[macOS](/platforms/macos-credentials#ci-zh)、[Windows](/platforms/windows#5-代码签名)。

::: danger
绝不要提交密钥库、`.p12` / `.p8` 文件、证书或密码。一旦泄露，就视为已被攻破：修改密码、吊销 API 密钥，或创建新证书。
:::

## 下一步

[钩子](/guides/hooks)。
