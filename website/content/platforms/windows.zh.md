---
title: "Windows"
description: "构建便携 zip、Inno Setup 安装程序或 MSIX 包，并进行签名。"
---

本教程构建 Windows 应用：便携 zip（`windows`）、安装程序（`exe`，使用 Inno Setup）和 `.msix` 包（`msix`），可选代码签名。只能在 Windows 上构建；三个目标共用一次 `flutter build windows`。

## 你需要什么

- 装有 *使用 C++ 的桌面开发* 工作负载的 **Visual Studio**。
- `exe`：[Inno Setup 6](https://jrsoftware.org/isinfo.php)（`winget install --id JRSoftware.InnoSetup -e`）。fpack 会在默认安装目录找 `ISCC.exe`，也可用 `windows.inno_setup.iscc` 指定。
- `msix`：把 `msix` 包作为开发依赖：`dart pub add --dev msix`。
- 签名：`signtool`（Windows SDK）和代码签名证书。
- 项目路径**只能包含 ASCII 字符**。`flutter build windows` 在带有中文等字符的路径下会失败（`Unable to read file … app.dill`），doctor 会提示。也可以在 Windows 区域设置里启用“Beta: 使用 Unicode UTF-8 提供全球语言支持”。

## 1. 检查机器

```powershell
fpack doctor windows exe msix
```

## 2. 构建便携 zip

```powershell
fpack build windows
```

结果是 `dist\1.0.0+1\my_app-1.0.0+1-windows-x64-portable.zip`：压缩后的 Release 目录。用户解压后运行 `.exe` 即可。

## 3. 构建安装程序

```powershell
fpack build exe
```

fpack 根据项目元数据生成 Inno Setup 脚本，编译成 `my_app-1.0.0+1-windows-x64-setup.exe`。`fpack init` 只生成一次 `windows.inno_setup.app_id`，请**保持它不变**，这样升级才会替换已安装的版本。

```yaml
windows:
  inno_setup:
    languages: [en, zh-CN]      # 第一个是默认语言；fpack 自带旧版 Inno 缺少的中文语言文件
    privileges: user            # user（无需管理员）| admin | ask
    desktop_icon: checked       # checked | unchecked | none
    license_file: LICENSE.txt
```

Inno Setup 还支持发布者和网址、版权、安装目录、开始菜单组、信息页、安装图标、向导图片、最低 Windows 版本和压缩方式（见[参考](/configuration#windowsinno_setup)）。要用自己的脚本，设置 `windows.inno_setup.script`。公司和产品信息来自 `app.*`（[名称、版本与输出](/guides/project-and-output)）。

## 4. 构建 MSIX

```powershell
fpack build msix
```

fpack 用 `windows.msix` 的设置运行 `dart run msix:create`，这些设置优先于 `pubspec.yaml` 里的 `msix_config`：

```yaml
windows:
  msix:
    publisher: "CN=Your Company"
    capabilities: [internetClient]
    protocol_activation: [myapp]
```

显示名称、发布者显示名称、标识名称、版本、徽标、语言、文件扩展名、执行别名、开机启动、最低系统版本和 Store 模式都可以设置。

::: note
`msix` 包只从 `pubspec.yaml` 读取描述（`msix_config.description`，否则是 `description`）。当 `windows.msix.description` 或 `app.description` 与之不同时，fpack 会打印需要添加的条目。
:::

## 5. 代码签名

代码签名是可选的，但能避免 SmartScreen 警告。在 `windows.sign` 里**选择一种**证书来源：

| 来源 | 键 | 变量 |
| --- | --- | --- |
| `.pfx` 文件 | `certificate` + `password` | `FPACK_WINDOWS_CERTIFICATE`、`FPACK_WINDOWS_CERTIFICATE_PASSWORD` |
| 证书库 | `thumbprint`（`/sha1`）或 `subject`（`/n`），配合 `store`（默认 `My`）和 `machine_store: true` | `FPACK_WINDOWS_CERT_THUMBPRINT`、`FPACK_WINDOWS_CERT_SUBJECT` |
| signtool 插件（云签名，如 Azure Trusted Signing） | `dlib` + `dlib_metadata` | `FPACK_WINDOWS_SIGN_DLIB`、`FPACK_WINDOWS_SIGN_DLIB_METADATA` |

```yaml
windows:
  sign:
    certificate: ${WINDOWS_PFX}            # 或 thumbprint: / subject: / dlib:
    password: ${WINDOWS_PFX_PASSWORD}
    url: https://example.com
```

`flutter build windows` 一结束，fpack 就对 Release 目录里的 `files`（默认 `*.exe` 和 `*.dll`）签名，所以便携 zip、安装程序和 MSIX 里的二进制都是已签名的。Inno Setup 的安装程序和卸载程序通过 `SignTool=` 签名；MSIX 由 `msix:create` 用同一张证书签名（使用 `dlib` 时，fpack 对生成好的 MSIX 签名）。每个签名都使用 SHA-256 和 RFC 3161 时间戳（`timestamp_url`，默认 `http://timestamp.digicert.com`），并带上 `description`（`/d`）和 `url`（`/du`）。之后 `signtool verify /pa` 会检查签名；失败（例如自签名的测试证书）只是警告，`verify: false` 可以关闭检查。

signtool 从 `PATH` 或最新的 Windows SDK 中查找，也可以设置 `signtool`。`fpack doctor` 会报告它以及安装方法：`winget install Microsoft.WindowsSDK.10.0.26100`（需要管理员；没有管理员权限时，从 NuGet 包 `Microsoft.Windows.SDK.BuildTools` 里提取 signtool 并设置 `signtool`）。

::: tip
把证书密码放在变量（`${WINDOWS_PFX_PASSWORD}`）或 CI 密钥里。fpack 绝不会打印它。
:::

## 钩子与引号

[钩子](/guides/hooks)在 Windows 上通过 `cmd /S /C` 运行，所以钩子变量要写成 `%VAR%`（`%FPACK_OUTPUT_DIR%`）；带引号的路径照常使用。

## 在 CI 中

使用 `windows-latest`；见 [CI 示例](/ci#windows-zh)。

## 出问题时

| 提示 | 处理 |
| --- | --- |
| `Inno Setup (ISCC.exe) not found` | `winget install --id JRSoftware.InnoSetup -e`，或设置 `windows.inno_setup.iscc` |
| `Unable to read file … app.dill` | 把项目移到只含 ASCII 的路径 |
| msix 目标提示缺少包 | `dart pub add --dev msix` |
| 找不到 signtool | 安装 Windows SDK 或设置 `windows.sign.signtool` |

更多见[故障排查页](/troubleshooting#windows-zh)。
