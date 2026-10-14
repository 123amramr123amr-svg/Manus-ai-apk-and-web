plugins {
    id("com.android.application")
}

// Optional: bake a default server URL into the APK with
//   gradle assembleDebug -PmanusServerUrl=https://your-app.example.com
val defaultServerUrl: String = (project.findProperty("manusServerUrl") as String?) ?: ""

android {
    namespace = "im.manus.webview"
    compileSdk = 34

    defaultConfig {
        applicationId = "im.manus.webview"
        minSdk = 24
        targetSdk = 34
        versionCode = ((project.findProperty("manusVersionCode") as String?) ?: "1").toInt()
        versionName = (project.findProperty("manusVersionName") as String?) ?: "1.0.0"
        buildConfigField("String", "DEFAULT_SERVER_URL", "\"$defaultServerUrl\"")
    }

    buildFeatures {
        buildConfig = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildTypes {
        getByName("release") {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }
}

// Release signing is optional and only enabled when the keystore environment
// variables are present (see docs/BUILD_APK.md). Debug builds are signed with
// the standard Android debug key and install on any device.
val keystorePath: String? = System.getenv("MANUS_KEYSTORE_PATH")
if (!keystorePath.isNullOrBlank()) {
    android.signingConfigs {
        create("release") {
            storeFile = file(keystorePath)
            storePassword = System.getenv("MANUS_KEYSTORE_PASSWORD")
            keyAlias = System.getenv("MANUS_KEY_ALIAS")
            keyPassword = System.getenv("MANUS_KEY_PASSWORD")
        }
    }
    android.buildTypes.getByName("release").signingConfig = android.signingConfigs.getByName("release")
}
