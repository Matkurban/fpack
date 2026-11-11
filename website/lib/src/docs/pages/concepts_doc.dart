import '../doc_node.dart';
import '../doc_page.dart';
import '../l10n.dart';

/// Concepts documentation page, authored in pure Dart with full internationalization.
class ConceptsDoc extends DocPage {
  const ConceptsDoc();

  @override
  String get path => 'concepts';

  @override
  L10n<String> get title => t(
        en: 'How fpack works',
        zh: 'fpack 如何工作',
      );

  @override
  L10n<String> get description => t(
        en: 'Targets, the build pipeline, where files go and what fpack never touches. Read this once; everything else builds on it.',
        zh: '目标、构建流程、文件放在哪里，以及 fpack 绝不会动什么。先读一遍，后面的内容都建立在这里。',
      );

  @override
  List<DocNode> get nodes => [
        DocParagraph(t(
          en: 'fpack is a packaging tool, not a build system. It drives the tools you already use (`flutter`, Gradle, Xcode, signtool, `dpkg-deb`, …) in the right order, adds signing, notarization and packaging, and collects the results. This page explains the few ideas the rest of the documentation relies on.',
          zh: 'fpack 是打包工具，不是构建系统。它按正确的顺序驱动你已经在用的工具（`flutter`、Gradle、Xcode、signtool、`dpkg-deb` 等），补上签名、公证和打包，并把结果收集起来。本页解释后面文档会反复用到的几个概念。',
        )),
        DocHeading(t(en: 'Targets and artifacts', zh: '目标与产物')),
        DocParagraph(t(
          en: 'A **target** is one kind of file you want to ship. You name targets on the command line:',
          zh: '**目标（target）**是你想发布的一种文件。在命令行里写出目标名：',
        )),
        DocCode(
          language: 'bash',
          t(
            en: 'fpack build apk aab       # two targets\nfpack build macos dmg pkg # three targets',
            zh: 'fpack build apk aab       # 两个目标\nfpack build macos dmg pkg # 三个目标',
          ),
        ),
        DocTable(
          headers: [
            t(en: 'Platform', zh: '平台'),
            t(en: 'Targets', zh: '目标'),
            t(en: 'What you get', zh: '得到什么'),
          ],
          rows: [
            [
              literal('Android'),
              literal('`apk`, `aab`'),
              t(
                en: 'APK (universal and/or per ABI), Android App Bundle',
                zh: 'APK（通用和/或按 ABI 拆分）、Android App Bundle',
              ),
            ],
            [
              literal('iOS'),
              literal('`ipa`'),
              t(
                en: 'IPA for App Store Connect, ad hoc, enterprise, or unsigned',
                zh: '用于 App Store Connect、Ad Hoc、企业分发或未签名的 IPA',
              ),
            ],
            [
              literal('macOS'),
              literal('`macos`, `dmg`, `pkg`'),
              t(
                en: 'zipped `.app`, disk image, installer package (signed, notarized)',
                zh: '压缩的 `.app`、磁盘映像、安装包（已签名、已公证）',
              ),
            ],
            [
              literal('Windows'),
              literal('`windows`, `exe`, `msix`'),
              t(
                en: 'portable zip, Inno Setup installer, MSIX',
                zh: '便携 zip、Inno Setup 安装程序、MSIX',
              ),
            ],
            [
              literal('Linux'),
              literal('`linux`, `deb`, `rpm`, `appimage`, `snap`'),
              t(
                en: 'tar.gz, Debian package, RPM, AppImage, Snap',
                zh: 'tar.gz、Debian 包、RPM、AppImage、Snap',
              ),
            ],
            [
              literal('Web'),
              literal('`web`'),
              t(
                en: 'zip of the deployable site',
                zh: '可部署站点的 zip',
              ),
            ],
          ],
        ),
        DocParagraph(t(
          en: 'An **artifact** is a file a target produced. Details of every target are on [Targets](/targets).',
          zh: '**产物（artifact）**是目标生成的文件。每个目标的详细信息见[目标与产物](/targets)。',
        )),
        DocCallout.note(t(
          en: 'Flutter cannot cross-compile desktop and iOS apps. You build iOS and macOS on a Mac, Windows targets on Windows and Linux targets on Linux. Android and web build anywhere. [`fpack build --all`](/guides/many-targets) builds everything *this* machine can build and skips the rest with a reason; naming a target the machine cannot build is an error that points you to a CI runner of that OS.',
          zh: 'Flutter 不能交叉编译桌面和 iOS 应用：iOS 和 macOS 要在 Mac 上构建，Windows 目标要在 Windows 上构建，Linux 目标要在 Linux 上构建；Android 和 Web 在哪里都能构建。[`fpack build --all`](/guides/many-targets) 会构建*这台*机器能构建的全部目标，其余的注明原因并跳过；明确指定一个本机无法构建的目标会报错，并指引你使用对应系统的 CI 机器。',
        )),
        DocHeading(t(en: 'The build pipeline', zh: '构建流程')),
        DocParagraph(t(
          en: 'For every run fpack does the same steps, so failures happen early and are easy to place:',
          zh: '每次运行 fpack 都执行同样的步骤，所以失败总是发生得早、也容易定位：',
        )),
        DocList(
          ordered: true,
          [
            t(
              en: '**Read** the project (`pubspec.yaml`, `android/app/build.gradle`, the Xcode project) and `fpack.yaml` if there is one. Invalid configuration stops here, with the line and a suggestion.',
              zh: '**读取**项目（`pubspec.yaml`、`android/app/build.gradle`、Xcode 工程）和 `fpack.yaml`（如果有）。配置无效会在这里停下，并给出行号和建议。',
            ),
            t(
              en: '**Check** each target (the same checks as [`fpack doctor`](/commands#fpack-doctor)): tools, SDKs, certificates, keystore passwords, flavors. Targets that cannot be built are reported *before* anything slow starts.',
              zh: '**检查**每个目标（与 [`fpack doctor`](/commands#fpack-doctor-zh) 相同）：工具、SDK、证书、密钥库密码、flavor。无法构建的目标会在任何耗时步骤开始*之前*报告。',
            ),
            t(
              en: '**Build** with Flutter once per platform. `macos` + `dmg` + `pkg` share one `flutter build macos`; `linux` + `deb` + `rpm` + `appimage` + `snap` share one `flutter build linux`.',
              zh: '**构建**：每个平台只运行一次 Flutter 构建。`macos` + `dmg` + `pkg` 共用一次 `flutter build macos`；`linux` + `deb` + `rpm` + `appimage` + `snap` 共用一次 `flutter build linux`。',
            ),
            t(
              en: '**Package and sign** each target: copy or repack, sign, notarize, verify.',
              zh: '**打包与签名**每个目标：复制或重新打包、签名、公证、验证。',
            ),
            t(
              en: '**Checksum and summarize**: write `SHA256SUMS` next to the artifacts and print a table of what was produced.',
              zh: '**校验和与汇总**：在产物旁写入 `SHA256SUMS`，并打印产出文件的表格。',
            ),
          ],
        ),
        DocParagraph(t(
          en: '`--dry-run` stops after step 2 and prints the exact commands steps 3 and 4 would run.',
          zh: '`--dry-run` 在第 2 步之后停下，打印第 3、4 步将运行的确切命令。',
        )),
        DocHeading(t(en: 'Where files go', zh: '文件放在哪里')),
        DocCode(
          t(
            en: '''my_app/
  dist/1.0.0+1/            artifacts and SHA256SUMS (output.dir)
  build/fpack/logs/        full log of every command
  build/fpack/secrets/     temporary key material, mode 0600, deleted after the build
  build/fpack/…            generated ExportOptions.plist, snapcraft.yaml, …''',
            zh: '''my_app/
  dist/1.0.0+1/            产物和 SHA256SUMS（output.dir）
  build/fpack/logs/        每条命令的完整日志
  build/fpack/secrets/     临时密钥文件，权限 0600，构建后删除
  build/fpack/…            生成的 ExportOptions.plist、snapcraft.yaml 等''',
          ),
        ),
        DocList([
          t(
            en: '**Artifacts** go to `dist/<version>+<build>/`. File names follow a template ([names and output](/guides/project-and-output)). Existing artifacts are never overwritten unless you pass `--force`: bump the version, pass `--build-number N`, or force it.',
            zh: '**产物**放在 `dist/<版本>+<构建号>/`。文件名遵循模板（[名称与输出](/guides/project-and-output)）。默认**绝不覆盖**已有产物：提高版本号、传 `--build-number N`，或者用 `--force`。',
          ),
          t(
            en: '**Everything else** fpack creates is under `build/fpack/` and can be deleted with [`fpack clean`](/commands#fpack-clean).',
            zh: '**fpack 创建的其他一切**都在 `build/fpack/` 下，可以用 [`fpack clean`](/commands#fpack-clean-zh) 删除。',
          ),
          t(
            en: '**Your sources are read-only.** fpack does not edit Gradle files, Xcode projects, `pubspec.yaml`, icon or splash configs. Add `dist/` to `.gitignore`.',
            zh: '**你的源码是只读的。** fpack 不会修改 Gradle 文件、Xcode 工程、`pubspec.yaml`、图标或启动页配置。建议把 `dist/` 加入 `.gitignore`。',
          ),
        ]),
        DocHeading(t(en: 'Configuration: flags, variables, file', zh: '配置：参数、变量、文件')),
        DocParagraph(t(
          en: 'fpack needs no configuration. When you add some, it comes from four places; the first one that sets a value wins:',
          zh: 'fpack 不需要配置。需要时，配置有四个来源，先设置了值的那个生效：',
        )),
        DocList(
          ordered: true,
          [
            literal('1. command-line flags (`--flavor prod`)'),
            literal('2. `FPACK_*` environment variables (`FPACK_FLAVOR=prod`)'),
            literal('3. `fpack.yaml` in the project root (`build.flavor: prod`)'),
            t(
              en: '4. defaults (read from your project, or fpack\'s own)',
              zh: '4. 默认值（从你的项目读取，或使用 fpack 自己的默认）',
            ),
          ],
        ),
        DocParagraph(t(
          en: '`fpack init` writes an `fpack.yaml` that lists every key, commented out, with explanations; uncomment what you need. Inside the file, `\${VAR}` and `\${VAR:-default}` read environment variables, which is how secrets stay out of the file ([Secrets and variables](/guides/secrets)).',
          zh: '`fpack init` 会写出一个列出**所有**键的 `fpack.yaml`，全部注释掉并带说明，需要什么就取消注释什么。文件里的 `\${VAR}` 和 `\${VAR:-默认值}` 会读取环境变量，密钥就是这样不进文件的（[密钥与环境变量](/guides/secrets)）。',
        )),
        DocHeading(t(en: 'Safety nets', zh: '安全网')),
        DocTable(
          headers: [
            t(en: 'You want to know', zh: '你想知道'),
            t(en: 'Run', zh: '运行'),
          ],
          rows: [
            [
              t(
                en: 'What can this machine build, and what is missing?',
                zh: '这台机器能构建什么？缺什么？',
              ),
              literal('`fpack doctor` (or `fpack list`)'),
            ],
            [
              t(en: 'Which commands will run?', zh: '会运行哪些命令？'),
              literal('`fpack build … --dry-run`'),
            ],
            [
              t(
                en: 'Why did it fail?',
                zh: '为什么失败？',
              ),
              t(
                en: 'read the excerpt and fix fpack prints; the full log path is shown; add `-v` to stream tool output',
                zh: '看 fpack 打印的错误摘要和修复建议；会显示完整日志路径；加 `-v` 查看工具的完整输出',
              ),
            ],
            [
              t(
                en: 'What exactly was produced?',
                zh: '到底产出了什么？',
              ),
              literal('`dist/<version>/SHA256SUMS`, or `--json`'),
            ],
          ],
        ),
        DocHeading(t(en: 'Vocabulary', zh: '术语')),
        DocTable(
          headers: [
            t(en: 'Term', zh: '术语'),
            t(en: 'Meaning', zh: '含义'),
          ],
          rows: [
            [
              t(en: 'target', zh: '目标'),
              t(en: 'a kind of artifact: `apk`, `dmg`, `deb`, …', zh: '一种产物：`apk`、`dmg`、`deb` 等'),
            ],
            [
              literal('flavor'),
              t(
                en: 'an Android productFlavor / Xcode scheme ([Flavors](/guides/build-variants))',
                zh: 'Android productFlavor / Xcode scheme（[Flavor](/guides/build-variants)）',
              ),
            ],
            [
              t(en: 'mode', zh: '模式'),
              t(en: '`release` (default), `profile` or `debug`', zh: '`release`（默认）、`profile` 或 `debug`'),
            ],
            [
              t(en: 'launcher / core', zh: '启动器 / 核心'),
              t(
                en: 'the small Dart program you install, and the native binary it downloads and runs ([Installation](/installation#launcher))',
                zh: '你安装的小型 Dart 程序，以及它下载并运行的原生二进制（[安装](/installation#launcher-zh)）',
              ),
            ],
            [
              t(en: 'notarization', zh: '公证'),
              t(
                en: "Apple's malware scan for macOS software distributed outside the App Store ([macOS](/platforms/macos))",
                zh: 'Apple 对在 App Store 之外分发的 macOS 软件做的恶意软件扫描（[macOS](/platforms/macos)）',
              ),
            ],
          ],
        ),
        DocHeading(t(en: 'Next', zh: '下一步')),
        DocParagraph(t(
          en: 'Install fpack on [the next page](/installation), then follow [your first build](/getting-started).',
          zh: '在[下一页](/installation)安装 fpack，然后进行[第一次构建](/getting-started)。',
        )),
      ];
}

