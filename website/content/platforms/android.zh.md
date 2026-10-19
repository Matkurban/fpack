---
title: "Android"
description: "构建已签名的 APK 和 AAB：密钥库、签名、按 ABI 拆分、验证与上架 Google Play。"
---

本教程带一个 Android 应用从“用 debug 密钥构建”走到可上架 Google Play 的已签名 `.aab`，以及可直接分发的已签名 `.apk`。目标：`apk`、`aab`。可在 macOS、Windows 和 Linux 上构建。

## 你需要什么

- Flutter、Android SDK（已接受许可）和 **Java 17 或更新版本**。Android Studio 全都自带。
- 由 Flutter 创建的 Android 工程（`android/` 目录）。fpack **绝不会修改你的 Gradle 文件**。

## 1. 检查机器

```bash
fpack doctor apk aab
```

应当看到 `✓ Android SDK`、`✓ Android SDK licenses accepted` 和一条 Java 记录。如果提示缺少 Java，安装 Android Studio 或 JDK 17/21，然后运行 `flutter config --jdk-dir <路径>`。没有配置签名时 doctor 还会警告：*release builds are signed with the DEBUG key … Google Play will reject it*，第 3 步就是解决它。

## 2. 先构建一次（debug 签名）

```bash
fpack build apk
```

结果是 `dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk`。它可以安装到设备和模拟器，但用的是 Android **debug** 密钥，所以 Google Play 不会接受。这一步用来在加入签名之前确认工具链能用。

## 3. 创建上传密钥库

密钥库保存用来给应用签名的密钥。创建一次，并妥善保管：

```bash
keytool -genkey -v -keystore ~/keys/upload.jks -keyalg RSA -keysize 2048 \
  -validity 10000 -alias upload
```

`keytool` 会询问密钥库密码、密钥密码和你的信息。

::: warning
把密钥库文件和两个密码备份到私密的地方（密码管理器、加密磁盘），并且**不要提交到 git**。启用 Google Play 应用签名后，你在这里创建的是*上传*密钥：Google 用它自己的密钥给用户分发的应用重新签名，上传密钥丢失后可以请求重置。不通过 Google Play 分发的应用，密钥丢了就无法找回。
:::

## 4. 告诉 fpack

fpack 通过 Android Gradle 插件标准的 `android.injected.signing.*` 属性（以 `ORG_GRADLE_PROJECT_*` 环境变量传入）为 release 构建签名。密码不会出现在命令行、日志或 `--dry-run` 输出中。

最简单的方式是环境变量：

::: tabs
@@ macOS / Linux
```bash
export FPACK_ANDROID_KEYSTORE=~/keys/upload.jks
export FPACK_ANDROID_KEYSTORE_PASSWORD='…'
export FPACK_ANDROID_KEY_ALIAS=upload
export FPACK_ANDROID_KEY_PASSWORD='…'       # 省略时与密钥库密码相同
```
@@ Windows (PowerShell)
```powershell
$env:FPACK_ANDROID_KEYSTORE = "$HOME\keys\upload.jks"
$env:FPACK_ANDROID_KEYSTORE_PASSWORD = '…'
$env:FPACK_ANDROID_KEY_ALIAS = 'upload'
$env:FPACK_ANDROID_KEY_PASSWORD = '…'       # 省略时与密钥库密码相同
```
:::

或者把非敏感部分写进 `fpack.yaml`，密码仍放在变量里：

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    key_password: ${FPACK_ANDROID_KEY_PASSWORD}
```

如果项目自己已有带 `signingConfigs` 的 `android/key.properties`，就不要配置 fpack 签名，直接使用项目自己的设置。

## 5. 构建发布文件

```bash
fpack build apk aab
```

Gradle 启动前，fpack 先用 `keytool` 检查密钥库：密码或别名错误会立即失败，并指出是哪一个。构建后它会验证产物（`apksigner`、`keytool -printcert`）并打印签名者；如果仍是 *Android Debug* 证书，会给出警告。

```text
dist/1.0.0+1/
  my_app-1.0.0+1-android-universal.apk
  my_app-1.0.0+1-android.aab
  SHA256SUMS
```

## 6. 自己验证（可选）

```bash
keytool -printcert -jarfile dist/1.0.0+1/my_app-1.0.0+1-android.aab
apksigner verify --verbose --print-certs dist/1.0.0+1/my_app-1.0.0+1-android-universal.apk
```

`apksigner` 在 `$ANDROID_HOME/build-tools/<版本>/` 里。

## 7. 上传到 Google Play

在 Play 管理中心上传 `my_app-…-android.aab`（发布 → 正式版或测试 → 创建版本）。Play 会根据 bundle 为每种设备生成优化后的 APK。每次上传都需要更大的构建号：提高 `pubspec.yaml` 中 `+` 后面的数字，或传 `--build-number 43`。

## APK 变体

### 按 ABI 拆分

```bash
fpack build apk --split-per-abi           # 每个 ABI 一个 APK，不再生成通用 APK
fpack build apk --split-per-abi=both      # 按 ABI 的 APK 加通用 APK
fpack build apk --abis arm64-v8a,x86_64   # 限制要构建的 ABI
```

按 ABI 拆分的 APK 小得多；通用 APK 在所有设备上都能运行。默认 ABI：`armeabi-v7a`、`arm64-v8a`、`x86_64`。在 `fpack.yaml` 里是 `android.split_per_abi`（`false`、`true`、`both`）和 `android.abis`；变量是 `FPACK_SPLIT_PER_ABI`。

### 签名方案

默认使用 Gradle 自己的签名。设置了 `android.signing.v1` … `v4` 中的任何一个，fpack 就会用 `apksigner` 重新给 APK 签名，由你控制签名方案：

```yaml
android:
  signing:
    store_file: ~/keys/upload.jks
    store_password: ${FPACK_ANDROID_KEYSTORE_PASSWORD}
    key_alias: upload
    v1: true     # JAR 签名，Android 7 以下需要
    v2: true     # Android 7+
    v3: true     # Android 9+，支持密钥轮换
    v4: true     # Android 11+ 增量安装；同时写出 <apk>.idsig
```

### Gradle 属性与额外参数

`android.project_args` 传递 Gradle 项目属性（`flutter build -P key=value`），你的 `build.gradle` 可以用 `project.findProperty` 读取，例如开关 R8 混淆。`android.extra_args` 给 `flutter build apk` / `appbundle` 追加参数：

```yaml
android:
  project_args: { minify: "true" }
  extra_args: [--android-skip-build-dependency-validation]
```

## Flavor

如果你的项目定义了 `productFlavors`，传 `--flavor prod`（或 `build.flavor`），见 [Flavor、入口与编译变量](/guides/build-variants#flavor)。没有 flavor 的项目永远不需要这个参数。

## 在 CI 中

把密钥库 base64 后放进密钥（secret），让 fpack 在运行时写出文件：

```bash
base64 -i ~/keys/upload.jks | pbcopy      # macOS；Linux：base64 -w0 upload.jks
```

设置 `FPACK_ANDROID_KEYSTORE_BASE64`、`FPACK_ANDROID_KEYSTORE_PASSWORD`、`FPACK_ANDROID_KEY_ALIAS`（密钥密码不同时再设置 `FPACK_ANDROID_KEY_PASSWORD`）。fpack 把密钥库写到 `build/fpack/secrets/`（权限 0600），构建后删除。完整工作流见 [CI 示例](/ci#android-zh)。

## 出问题时

| 提示 | 处理 |
| --- | --- |
| `no Java (JDK 17+) found` | 安装 Android Studio 或 JDK 17/21，然后 `flutter config --jdk-dir <路径>` |
| `keystore password was incorrect` / `alias … does not exist` | 检查 `FPACK_ANDROID_KEYSTORE_PASSWORD` / `FPACK_ANDROID_KEY_ALIAS` |
| `… defines productFlavors … but no flavor is set` | 传 `--flavor` 或设置 `build.flavor` |
| Google Play 拒绝上传（debug 证书） | 完成第 4 步；检查构建后打印的签名者 |

更多见[故障排查页](/troubleshooting#android-zh)。
