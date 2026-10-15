#!/usr/bin/env bash
# Lightweight APK builder that does not need Gradle or Android Studio.
#
# It runs the standard Android toolchain directly:
#   aapt2 compile/link -> javac -> d8 -> zipalign -> apksigner
#
# Requirements:
#   - JDK 17 (JAVA_HOME or javac on PATH)
#   - Android SDK with build-tools and a platform, e.g.
#       sdkmanager "platforms;android-34" "build-tools;34.0.0"
#     Point ANDROID_SDK_ROOT (or ANDROID_HOME) at the SDK directory.
#
# Usage:
#   ANDROID_SDK_ROOT=~/Android/Sdk ./android/build-manual.sh [output.apk] [default_server_url]
#
# Environment overrides:
#   BUILD_TOOLS_VERSION (default 34.0.0)   PLATFORM_VERSION (default android-34)
#   VERSION_NAME (1.0.0)                   VERSION_CODE (1)
#   MIN_SDK (24)                           TARGET_SDK (34)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$SCRIPT_DIR/app"
OUT_APK="${1:-$SCRIPT_DIR/../dist/manus-ai-webview-debug.apk}"
DEFAULT_SERVER_URL="${2:-}"

SDK="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
if [[ -z "$SDK" ]]; then
  echo "error: set ANDROID_SDK_ROOT (or ANDROID_HOME) to your Android SDK directory" >&2
  exit 1
fi

BUILD_TOOLS_VERSION="${BUILD_TOOLS_VERSION:-34.0.0}"
PLATFORM_VERSION="${PLATFORM_VERSION:-android-34}"
VERSION_NAME="${VERSION_NAME:-1.0.0}"
VERSION_CODE="${VERSION_CODE:-1}"
MIN_SDK="${MIN_SDK:-24}"
TARGET_SDK="${TARGET_SDK:-34}"
PACKAGE_NAME="im.manus.webview"

BUILD_TOOLS="$SDK/build-tools/$BUILD_TOOLS_VERSION"
ANDROID_JAR="$SDK/platforms/$PLATFORM_VERSION/android.jar"
WORK_DIR="$SCRIPT_DIR/.build-manual"
KEYSTORE="$SCRIPT_DIR/.debug.keystore"

for tool in "$BUILD_TOOLS/aapt2" "$BUILD_TOOLS/d8" "$BUILD_TOOLS/zipalign" "$BUILD_TOOLS/apksigner"; do
  [[ -x "$tool" ]] || { echo "error: missing $tool" >&2; exit 1; }
done
[[ -f "$ANDROID_JAR" ]] || { echo "error: missing $ANDROID_JAR" >&2; exit 1; }

JAVAC="${JAVA_HOME:+$JAVA_HOME/bin/}javac"
KEYTOOL="${JAVA_HOME:+$JAVA_HOME/bin/}keytool"

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"/{res,gen,classes,dex} "$(dirname "$OUT_APK")"

echo "==> compiling resources"
"$BUILD_TOOLS/aapt2" compile --dir "$APP_DIR/src/main/res" -o "$WORK_DIR/res/resources.zip"

# aapt2 needs a package attribute, while AGP 8 expects the namespace in
# build.gradle.kts only, so the manifest is patched for this standalone build.
MANIFEST="$WORK_DIR/AndroidManifest.xml"
sed '0,/<manifest/{s|<manifest|<manifest package="'"$PACKAGE_NAME"'"|}' \
  "$APP_DIR/src/main/AndroidManifest.xml" > "$MANIFEST"

echo "==> linking resources and manifest"
"$BUILD_TOOLS/aapt2" link \
  -o "$WORK_DIR/base.apk" \
  -I "$ANDROID_JAR" \
  --manifest "$MANIFEST" \
  -R "$WORK_DIR/res/resources.zip" \
  -A "$APP_DIR/src/main/assets" \
  --java "$WORK_DIR/gen" \
  --min-sdk-version "$MIN_SDK" \
  --target-sdk-version "$TARGET_SDK" \
  --version-code "$VERSION_CODE" \
  --version-name "$VERSION_NAME" \
  --auto-add-overlay

echo "==> generating BuildConfig"
mkdir -p "$WORK_DIR/gen/$PACKAGE_NAME"
cat > "$WORK_DIR/gen/$PACKAGE_NAME/BuildConfig.java" <<EOF
package $PACKAGE_NAME;

public final class BuildConfig {
    public static final boolean DEBUG = true;
    public static final String VERSION_NAME = "$VERSION_NAME";
    public static final String DEFAULT_SERVER_URL = "$DEFAULT_SERVER_URL";

    private BuildConfig() {
    }
}
EOF

echo "==> compiling Java sources"
mapfile -t JAVA_SOURCES < <(find "$APP_DIR/src/main/java" "$WORK_DIR/gen" -name '*.java')
"$JAVAC" -source 17 -target 17 -encoding UTF-8 -Xlint:-options \
  -classpath "$ANDROID_JAR" \
  -d "$WORK_DIR/classes" \
  "${JAVA_SOURCES[@]}"

echo "==> dexing"
mapfile -t CLASS_FILES < <(find "$WORK_DIR/classes" -name '*.class')
"$BUILD_TOOLS/d8" \
  --min-api "$MIN_SDK" \
  --lib "$ANDROID_JAR" \
  --output "$WORK_DIR/dex" \
  "${CLASS_FILES[@]}"

echo "==> packaging classes.dex"
python3 - "$WORK_DIR/base.apk" "$WORK_DIR/dex/classes.dex" <<'PY'
import shutil, sys, zipfile
apk, dex = sys.argv[1], sys.argv[2]
shutil.copyfile(apk, apk + ".tmp")
with zipfile.ZipFile(apk + ".tmp", "a", zipfile.ZIP_DEFLATED) as archive:
    archive.write(dex, "classes.dex")
shutil.move(apk + ".tmp", apk)
PY

echo "==> aligning"
"$BUILD_TOOLS/zipalign" -f -p 4 "$WORK_DIR/base.apk" "$WORK_DIR/aligned.apk"

if [[ ! -f "$KEYSTORE" ]]; then
  echo "==> creating a debug keystore"
  "$KEYTOOL" -genkeypair -keystore "$KEYSTORE" -alias androiddebugkey \
    -storepass android -keypass android -keyalg RSA -keysize 2048 -validity 10000 \
    -dname "CN=Android Debug,O=Android,C=US" >/dev/null 2>&1
fi

echo "==> signing"
"$BUILD_TOOLS/apksigner" sign \
  --ks "$KEYSTORE" --ks-pass pass:android --key-pass pass:android \
  --ks-key-alias androiddebugkey \
  --out "$OUT_APK" "$WORK_DIR/aligned.apk"

"$BUILD_TOOLS/apksigner" verify --print-certs "$OUT_APK" | head -5
echo "APK ready: $OUT_APK"
