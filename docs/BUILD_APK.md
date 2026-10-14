# بناء ملف APK — دليل كامل / Building the APK

<div dir="rtl">

## ما هو هذا التطبيق؟

مشروع أندرويد بسيط (Java، بدون مكتبات خارجية) يفتح تطبيق الويب في `WebView` ويتيح:

* إدخال عنوان الخادم وحفظه على الجهاز (أو تضمينه مسبقًا في البناء).
* اختيار الملفات من الجهاز لرفعها إلى Manus (منتقي ملفات أصلي).
* صفحة "لا يمكن الوصول" مدمجة مع زر إعادة المحاولة.
* جسر JS بسيط: `window.Android.getServerUrl()`, `setServerUrl(url)`, `openSettings()`, `retry()`, `toast(msg)`.

المتطلبات: JDK 17، Android SDK (compileSdk 34)، Gradle 8.7+.

</div>

---

## الطريقة 1: GitHub Actions (بدون أي إعداد محلي)

<div dir="rtl">

1. اذهب إلى **Actions → Build APK → Run workflow**.
2. يمكنك وضع رابط خادمك في الحقل `server_url` (اختياري) ليُدمج داخل التطبيق.
3. بعد انتهاء التشغيل: **Artifacts → manus-ai-apk → app-debug.apk**.
4. عند دفع وسم (`git tag v1.0.0 && git push origin v1.0.0`) يُرفق الـ APK في صفحة **Releases** تلقائيًا.

</div>

---

## الطريقة 2: بناء محلي بـ Gradle

```bash
cd android

# بناء نسخة تجريبية موقّعة بمفتاح التطوير (قابلة للتثبيت مباشرة)
gradle assembleDebug
# الناتج: app/build/outputs/apk/debug/app-debug.apk

# تضمين عنوان الخادم افتراضيًا
gradle assembleDebug -PmanusServerUrl=https://your-app.example.com

# نسخة إصدار (release) غير موقّعة
gradle assembleRelease
```

<div dir="rtl">

### التوقيع بمفتاح خاص (للنشر)

</div>

```bash
keytool -genkeypair -v -keystore release.jks -alias manus \
  -keyalg RSA -keysize 2048 -validity 10000 \
  -storepass "STRONG_PASSWORD" -keypass "STRONG_PASSWORD" \
  -dname "CN=Manus AI, OU=Mobile, O=Example, L=Riyadh, C=SA"

export MANUS_KEYSTORE_PATH="$PWD/release.jks"
export MANUS_KEYSTORE_PASSWORD="STRONG_PASSWORD"
export MANUS_KEY_ALIAS="manus"
export MANUS_KEY_PASSWORD="STRONG_PASSWORD"

gradle assembleRelease
# الناتج: app/build/outputs/apk/release/app-release.apk
```

<div dir="rtl">

في GitHub Actions يمكنك تمرير نفس المتغيرات كـ **Secrets** وسيستخدمها الـ workflow تلقائيًا
(راجع `android/app/build.gradle.kts`).

</div>

---

## الطريقة 3: Android Studio

<div dir="rtl">

1. `File → Open` واختر مجلد `android/`.
2. انتظر مزامنة Gradle (سيُنزّل Android SDK 34 تلقائيًا).
3. `Run` للتشغيل على جهاز/محاكي، أو `Build → Build APK(s)`.

</div>

---

## الطريقة 4: بدون APK — تثبيت كـ PWA

<div dir="rtl">

افتح رابط تطبيق الويب في Chrome على أندرويد ← قائمة النقاط الثلاث ← **تثبيت التطبيق / إضافة إلى الشاشة الرئيسية**.
سيُثبَّت بأيقونة وشاشة كاملة، ويعمل كتطبيق عادي (مع Service Worker للواجهة).

</div>

---

## استكشاف الأخطاء

| المشكلة | الحل |
| --- | --- |
| التطبيق يفتح صفحة "لا يمكن الوصول" | تحقق من عنوان الخادم، واضغط "عنوان الخادم" لتعديله. إن كان الخادم على `http://` محلي فاستخدم IP الجهاز وليس `localhost`. |
| الرفع لا يعمل | تأكد من منح صلاحية الوصول للملفات في منتقي الملفات (Android 13+ قد يطلب ذلك). |
| Gradle يفشل بسبب Java | استخدم JDK 17 (`java -version`) واضبط `JAVA_HOME`. |
| تريد أيقونة مختلفة | عدّل `tools/genicons/main.go` ثم `go run ./tools/genicons` (يحدّث أيقونات الويب والأندرويد). |
| الشاشة بيضاء | فعّل تصحيح WebView: `chrome://inspect` على جهاز مربوط بـ USB (مفعّل في نسخة debug). |
