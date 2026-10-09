package hints

import (
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	cases := map[string]string{
		"Keystore file '/x/upload.jks' not found for signing config 'release'.":           "android-keystore-missing",
		"> Keystore was tampered with, or password was incorrect":                         "android-keystore-password",
		"Failed to read key upload from store \"/x.jks\": Cannot recover key":             "android-keystore-password",
		"You have not accepted the license agreements of the following SDK components":    "android-licenses",
		"No Android SDK found. Try setting the ANDROID_HOME environment variable.":        "android-sdk-missing",
		"Unsupported class file major version 65":                                         "java-mismatch",
		"Android Gradle plugin requires Java 17 to run. You are currently using Java 11.": "java-mismatch",
		"Warning: CocoaPods not installed. Skipping pod install.":                         "cocoapods-missing",
		"CocoaPods's specs repository is too out-of-date to satisfy dependencies.":        "cocoapods-outdated",
		"Error running pod install":                                                       "cocoapods-failed",
		"error: Signing for \"Runner\" requires a development team.":                      "ios-signing-team",
		"No valid code signing certificates were found":                                   "ios-signing-team",
		"error: exportArchive: No profiles for 'com.x.app' were found":                    "ios-profile",
		"Encountered error while creating the IPA:":                                       "ios-export",
		"  status: Invalid": "notary-invalid",
		"productbuild: error: Could not find appropriate signing identity for “Developer ID Installer: Acme (TEAM123456)”.":           "pkg-identity",
		"Error: abortedUpload(resumeRequest: SotoS3.S3.ResumeMultipartUploadRequest(...), error: HTTPClientError.deadlineExceeded)":   "notary-network",
		"Error: No Keychain password item found for profile: XueHua":                                                                  "notary-auth",
		"hdiutil: create failed - Resource busy":                                                                                      "hdiutil-busy",
		"CMake Error: CMake was unable to find a build program corresponding to \"Ninja\".":                                           "linux-ninja",
		"Package 'gtk+-3.0', required by 'virtual:world', not found":                                                                  "linux-gtk",
		"Unable to find suitable Visual Studio toolchain.":                                                                            "windows-vs",
		"Because app depends on xue_hua_sdk from path which doesn't exist (could not find package xue_hua_sdk at \"../xue_hua_sdk\")": "pub-path-dep",
		"lib/main.dart:12:5: Error: Expected ';' after this.":                                                                         "dart-compile",
		"Missing classes detected while running R8.":                                                                                  "android-r8",
		"java.lang.OutOfMemoryError: Java heap space":                                                                                 "gradle-oom",
		"NDK at /sdk/ndk/27.0 did not have a source.properties file":                                                                  "android-ndk",
		"uses-sdk:minSdkVersion 21 cannot be smaller than version 24 declared in library":                                             "android-minsdk",
	}
	for line, want := range cases {
		h, ok := Match([]string{"some noise", line, "more noise"})
		if !ok || h.ID != want {
			t.Errorf("%q: got %q (ok=%v) want %q", line, h.ID, ok, want)
		}
		if h.Text() == "" {
			t.Errorf("%s has empty text", want)
		}
	}
	if _, ok := Match([]string{"all good", "Built build/app/outputs/flutter-apk/app-release.apk"}); ok {
		t.Error("unexpected match on success output")
	}
}

func TestExcerptGradle(t *testing.T) {
	lines := []string{"Running Gradle task 'assembleRelease'...", "", "FAILURE: Build failed with an exception.", "", "* What went wrong:",
		"Execution failed for task ':app:packageRelease'.", "> Keystore file not found", "", "* Try:", "> Run with --stacktrace", "BUILD FAILED in 3s"}
	ex := Excerpt(lines, 20)
	if !strings.Contains(ex[0], "What went wrong") || len(ex) != 3 {
		t.Fatalf("excerpt: %q", ex)
	}
}

func TestExcerptFallback(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "info line")
	}
	lines = append(lines, "lib/main.dart:3:1: Error: oops", "next", "Target kernel_snapshot failed")
	ex := Excerpt(lines, 10)
	if !strings.Contains(strings.Join(ex, "\n"), "Error: oops") {
		t.Fatalf("excerpt: %q", ex)
	}
	if len(Excerpt([]string{"a", "b"}, 10)) != 2 {
		t.Fatal("short")
	}
}
