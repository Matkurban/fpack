---
title: "Android"
description: "签名、flavor、APK 拆分与 App Bundle。"
lang: zh-CN
---

fpack **不修改 Gradle 文件**。它通过 Android Gradle 插件标准的 `android.injected.signing.*` 属性（以 `ORG_GRADLE_PROJECT_*` 环境变量传入）为 release 构建签名，密码不会出现在命令行、日志或 `--dry-run` 输出中（显示为 `***`）。

```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload-keystore.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'       # 省略时与 keystore 密码相同
fpack build apk aab
```

- CI 中可以把 keystore 本身 base64 后放进 `FPACK_ANDROID_KEYSTORE_BASE64`，fpack 会写到 `build/fpack/secrets/`（权限 0600），构建后删除。
- 构建前用 `keytool` 校验密码和别名，错误会直接指出是密码错还是别名不存在。
- 构建后用 `apksigner` / `keytool -printcert` 校验产物签名并显示签名者；如果仍是 Android Debug 证书会给出警告。
- 如果项目自己已有 `key.properties` + `signingConfigs`，不配置 fpack 签名即可，fpack 会使用项目的配置。
- 未配置签名且 release 使用 debug 签名（`flutter create` 的默认值）时会提示：可用于测试，Google Play 会拒绝。

## 创建上传密钥库

```bash
keytool -genkey -v -keystore ~/keys/upload.jks -keyalg RSA -keysize 2048 \
  -validity 10000 -alias upload
base64 -i ~/keys/upload.jks | pbcopy     # 用作 CI 密钥 FPACK_ANDROID_KEYSTORE_BASE64
```

不要把密钥库和密码提交到 git。启用 Google Play 应用签名时，Google 会用应用签名密钥重新签名；上面的密钥库是*上传*密钥。

## Flavor

`--flavor 名称`（或 `build.flavor`、`FPACK_FLAVOR`）选择 Android productFlavor。fpack 从 `android/app/build.gradle(.kts)` 读取 flavor：名称不存在时报错并给出建议；项目定义了 flavor 但没有指定时，fpack 会在完整 Gradle 构建失败之前就停止。产物文件名包含 flavor（`{-flavor}`）。

只存在于 Android 的 flavor，在 Xcode 工程没有自定义 scheme 时不会传给 iOS/macOS 构建。

## APK 拆分与 ABI

```bash
fpack build apk --split-per-abi           # 每个 ABI 一个 APK
fpack build apk --split-per-abi=both      # 按 ABI 拆分 + 通用 APK
fpack build apk --abis arm64-v8a,x86_64   # 限定 ABI
```

## App Bundle

`fpack build aab` 生成 `-android.aab`；构建后用 `keytool -printcert -jarfile` 校验签名者。
