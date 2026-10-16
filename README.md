# Manus AI — تطبيق ويب + APK (مبني على Manus AI Go SDK)

<div dir="rtl">

هذا المستودع يحوّل مكتبة [manus-ai-go](https://github.com/tigusigalpa/manus-ai-go) (وهي مكتبة Go للتعامل مع Manus API v2)
إلى **تطبيق جاهز للاستخدام**:

1. **تطبيق ويب كامل** (خادم Go + واجهة عربية/إنجليزية RTL) يعمل على أي استضافة تدعم Docker.
2. **تطبيق أندرويد (APK)** على شكل WebView يفتح نفس التطبيق، مع رفع ملفات أصلي (اختيار الملفات من الجهاز).
3. **PWA**: يمكن تثبيت الويب مباشرة على الشاشة الرئيسية من المتصفح بدون بناء APK.

</div>

---

## ⭐ الوضع المستقل: وكيل ذكاء اصطناعي داخل التطبيق (بدون أي خادم)

<div dir="rtl">

نسخة [OpenManus](https://github.com/FoundationAgents/OpenManus) المفتوحة المصدر **لا يمكن تشغيلها داخل APK**
(تحتاج Python 3.12 + متصفح Playwright + صندوق Docker، وكلها غير متوفرة على أندرويد — التفاصيل في
[`docs/AGENT_APP.md`](docs/AGENT_APP.md)). لذلك يحتوي هذا المستودع على **إعادة بناء كاملة للفكرة** داخل التطبيق:

* تطبيق يعمل **مستقلًا تمامًا** — بدون خادم وبدون مفاتيح Manus.
* تحتاج فقط **مفتاح LLM خاص بك**: OpenAI / DeepSeek / OpenRouter / Groq / Gemini / Mistral / xAI / Ollama / أي مزوّد متوافق.
* أدوات حقيقية تعمل بلا مفاتيح إضافية: **بحث ويب**، **قراءة صفحات**، ويكيبيديا، حاسبة، وقت، ملاحظات.
* حلقة وكيل (تخطيط → أداة → إجابة) مع عرض كل خطوة، إيقاف، تصدير المحادثة، محادثات محفوظة، واجهة عربية RTL.
* المفاتيح والمحادثات تبقى **على جهازك**، والطلبات تذهب مباشرة إلى مزوّد الموديل.

</div>

```text
تشغيل سريع:
1) ثبّت الـ APK → سيفتح على «الوضع المستقل» مباشرة
2) ⚙ الإعدادات → اختر المزوّد → ألصق مفتاحك → «اختبار الاتصال»
3) اكتب مهمتك: «ابحث عن … ولخّصها في 5 نقاط»
```

<div dir="rtl">

يمكنك أيضًا فتح نفس الواجهة من المتصفح على المسار `/agent/` بعد نشر خادم Go، أو تبديل الوضع من الإعدادات
إلى **الاتصال بخادم** إذا نشرت OpenManus/Manus على سيرفر خاص بك.

</div>

---

## ⚠️ وضع الخادم: عند أول تشغيل يسألك عن «عنوان الخادم»

<div dir="rtl">

هذا يظهر **فقط في وضع الخادم** (وليس في الوضع المستقل). الـ APK هنا واجهة أمام خادم تطبيق الويب (تطبيق Go في هذا المستودع)
لأن `api.manus.ai` لا يسمح باتصال مباشر من المتصفح/WebView (فحص CORS يمنعه).

**الحل الأسهل — نشر مجاني بنقرة واحدة ثم اكتب الرابط في الخانة:**

</div>

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/123amramr123amr-svg/Manus-ai-apk-and-web)

```text
مثال لما تكتبه في خانة عنوان الخادم:
https://manus-ai-web.onrender.com        ← بعد النشر على Render (الحل 2)
http://192.168.1.5:8080                  ← تشغيل الخادم على حاسوبك (الحل 1)
```

<div dir="rtl">

الدليل الكامل خطوة بخطوة (تشغيل محلي / نشر سحابي / PWA + حل المشاكل): **[`docs/FIRST_RUN.md`](docs/FIRST_RUN.md)**.

</div>

---

---

## 🐴 إضافة: تشغيل WormAgent (بوابة وكيل + واجهة تحكم + تلجرام)

<div dir="rtl">

[`BlackXV2vip/wormagent`](https://github.com/BlackXV2vip/wormagent) نسخة **مبنية مسبقًا** من بوابة
WormAgent `2026.3.19` (بوابة وكيل متعددة القنوات، Node/TS، رخصة MIT). تحتوي المشغّل + `dist/` + 75 إضافة،
**ولا تحتوي أي مصدر أندرويد أو ملف APK**.

</div>

```bash
# تشغيل محلي (بوابة + واجهة Control UI على 127.0.0.1:18789)
bash deploy/wormagent/setup.sh

# مع قناة تلجرام (مفتاح من @BotFather) وفتح الوصول للشبكة المحلية
bash deploy/wormagent/setup.sh --telegram "123456:AA..." --bind lan
```

<div dir="rtl">

* الواجهة على `http://127.0.0.1:18789/` (نفس واجهة المشروع) — ومن الجوال: أضفها للشاشة الرئيسية كـ PWA.
* تلجرام مُختبَر: القناة تتصل بـ `api.telegram.org` ولا ينقص إلا مفتاح BotFather.
* المحادثة تحتاج مفتاح مزوّد موديل: `node wormagent.mjs onboard`.
* التفاصيل الكاملة (تلجرام، الـ APK، الأمان، حل المشاكل): [`deploy/wormagent/README.md`](deploy/wormagent/README.md).

</div>


## المحتويات / Contents

| المسار | الوصف |
| --- | --- |
| `cmd/manusweb/` | نقطة تشغيل خادم الويب (`go run ./cmd/manusweb`) |
| `internal/app/` | الـ API: مهام، رسائل، ملفات، رصيد، وكلاء، مشاريع، Webhooks |
| `web/static/` | واجهة لوحة Manus (HTML/CSS/JS) + PWA، مُدمجة داخل الملف التنفيذي |
| `web/agent/` | **الوكيل المستقل**: واجهة + حلقة وكيل + أدوات، تعمل بدون خادم (تُقدَّم على `/agent/` و داخل الـ APK) |
| `tools/agent-harness/` | 19 اختبارًا لمنطق الوكيل (Node، بدون متصفح) |
| `tools/syncassets/` | نسخ `web/agent` إلى أصول الأندرويد (مع اختبار يمنع الاختلاف) |
| `third_party/manus-ai-go/` | نسخة مضَمَّنة من مكتبة Manus AI Go SDK (رخصة MIT) |
| `android/` | مشروع أندرويد (WebView، وضعان: مستقل + خادم) لبناء الـ APK |
| `tools/genicons/` | توليد أيقونات التطبيق (ويب + أندرويد) بدون ملفات ثنائية يدوية |
| `ci/github-workflows/` (تُنسخ إلى `.github/workflows/`) | CI للويب + بناء الـ APK تلقائيًا |
| `deploy/wormagent/` | **WormAgent**: سكربت تشغيل + دليل عربي (بوابة + Control UI + تلجرام) — انظر [`deploy/wormagent/README.md`](deploy/wormagent/README.md) |
| `docs/` | أدلة النشر والبناء ومرجع الـ API وتطبيق الوكيل |

---

## 1) تشغيل تطبيق الويب

<div dir="rtl">

### أسرع طريقة (بدون Docker)

</div>

```bash
# Go 1.22+ required
go run ./cmd/manusweb
# → http://localhost:8080
```

<div dir="rtl">

افتح المتصفح، اضغط ⚙ الإعدادات، وألصق **مفتاح Manus API** (يُحفظ في جهازك فقط).
بدلًا من ذلك يمكنك وضع المفتاح على الخادم مرة واحدة:

</div>

```bash
export MANUS_API_KEY="sk-your-manus-key"
export PORT=8080
go run ./cmd/manusweb
```

<div dir="rtl">

### باستخدام Docker

</div>

```bash
docker build -t manus-ai-web .
docker run --rm -p 8080:8080 -e MANUS_API_KEY="sk-..." manus-ai-web
# أو
docker compose up --build
```

<div dir="rtl">

### النشر السحابي

* **Render**: المستودع يحتوي `render.yaml` — أنشئ Blueprint واختر المستودع (يمكن ترك `MANUS_API_KEY` فارغًا ليدخل كل مستخدم مفتاحه).
* **Fly.io / Railway / VPS**: استخدم `Dockerfile` مباشرة؛ الخادم يقرأ متغيّر `PORT` تلقائيًا.
* بعد النشر، افتح الرابط، أضف المفتاح، ثم جرّب زر **اختبار الاتصال** في الإعدادات.

</div>

---

## 2) بناء ملف APK

<div dir="rtl">

لديك ثلاث طرق، من الأسهل للأصعب:

</div>

### الطريقة (أ) — GitHub Actions (الأسهل)

<div dir="rtl">

1. انسخ ملفات سير العمل إلى مكانها (مرة واحدة): `mkdir -p .github/workflows && cp ci/github-workflows/*.yml .github/workflows/` ثم `git push`
   (أو أنشئها من واجهة GitHub: **Add file → Create new file** بالمسار `.github/workflows/build-apk.yml` والصق المحتوى).
2. اذهب إلى تبويب **Actions** ← **Build APK** ← **Run workflow**.
3. (اختياري) أدخل رابط خادمك في `server_url` ليُدمج داخل التطبيق.
4. بعد انتهاء التشغيل، حمّل الملف من **Artifacts** باسم `manus-ai-apk`.
5. عند دفع وسم إصدار (`git tag v1.0.0 && git push --tags`) يُرفق الـ APK تلقائيًا في **Releases**.

</div>

### الطريقة (ب) — Android Studio

<div dir="rtl">

افتح مجلد `android/` في Android Studio (Gradle 8.7 + JDK 17)، ثم `Build → Build Bundle(s)/APK(s) → Build APK(s)`.
لتضمين عنوان الخادم افتراضيًا أضف في `android/gradle.properties`:

</div>

```properties
manusServerUrl=https://your-app.example.com
```

### الطريقة (ج) — سطر الأوامر / PWA

```bash
cd android
gradle assembleDebug            # الناتج: app/build/outputs/apk/debug/app-debug.apk
```

<div dir="rtl">

أو **بدون Gradle وبدون Android Studio** (aapt2 + javac + d8 + apksigner مباشرة، أسرع بكثير):

</div>

```bash
export ANDROID_SDK_ROOT=~/Android/Sdk
./android/build-manual.sh                       # → dist/manus-ai-webview-debug.apk
./android/build-manual.sh dist/app.apk https://your-app.example.com   # مع عنوان الخادم
```

<div dir="rtl">

أو بدون APK إطلاقًا: افتح رابط الويب في Chrome على أندرويد ← قائمة المتصفح ← **إضافة إلى الشاشة الرئيسية**
(التطبيق مثبّت كـ PWA ويعمل بشاشة كاملة).

التفاصيل الكاملة والتوقيع بمفتاح خاص: [`docs/BUILD_APK.md`](docs/BUILD_APK.md).

</div>

---

## 3) الإعدادات (متغيرات البيئة)

| المتغير | الافتراضي | الوصف |
| --- | --- | --- |
| `PORT` | `8080` | منفذ الخادم |
| `MANUS_API_KEY` | فارغ | مفتاح على مستوى الخادم (اختياري) |
| `MANUS_API_BASE_URL` | `https://api.manus.ai` | لتوجيه الطلبات عبر وسيط أو بيئة اختبار |
| `MANUS_WEBHOOK_TOKEN` | فارغ | إن حُدِّد، تُرفض أحداث الـ Webhook بدون `X-Webhook-Token` أو `?token=` |
| `MANUS_MAX_UPLOAD_MB` | `25` | أقصى حجم للملف المرفوع |
| `MANUS_EVENT_BUFFER` | `200` | عدد أحداث الـ Webhook المحفوظة في الذاكرة |
| `APP_VERSION` | `dev` | يظهر في `/api/health` و `/api/config` |

<div dir="rtl">

> **أمان:** المفتاح المُدخل من المتصفح يُخزَّن في `localStorage` ويُرسل إلى الخادم في الترويسة `X-Manus-Api-Key` لكل طلب.
> الخادم لا يسجّل المفاتيح ولا يحتفظ بها. إن كنت تنشر التطبيق للعامة، فضّل عدم وضع مفتاح في `MANUS_API_KEY`.

</div>

---

## 4) ماذا يمكنك أن تفعل داخل التطبيق؟

<div dir="rtl">

* **محادثة**: إنشاء مهمة جديدة (طلب، عنوان، ملف الوكيل `manus-1.6*`، اللغة، المشروع، الوضع التفاعلي)، متابعة الرسائل تلقائيًا كل 4 ثوانٍ، إرسال رسائل إضافية، إيقاف المهمة.
* **المهام**: قائمة المهام مع الحالة والرصيد المستهلك، إعادة التسمية، الحذف، الفتح في المحادثة.
* **الرصيد**: إجمالي/المجاني/الدوري/الإضافي، موعد التجديد، وسجل الاستخدام.
* **الأدوات**: المشاريع (مع إنشاء مشروع)، الوكلاء، المهارات، الموصّلات، متصفحات Manus المتصلة، وإدارة الـ Webhooks.
* **النشاط**: سجل مباشر لأحداث الـ Webhook الواردة (تم الإنشاء / اكتمل / توقف / ينتظر إدخالًا).
* **إرفاق ملفات**: من المتصفح أو من الأندرويد (منتقي ملفات أصلي)، مع رفع الملف إلى Manus ثم إرفاقه بـ `file_id`.
* **تعدد اللغات**: عربي (RTL) وإنجليزي بضغطة زر.

</div>

---

## 5) مرجع الـ API

<div dir="rtl">

كل المسارات تحت `/api` وتُرجع JSON. يُقرأ المفتاح من الترويسة `X-Manus-Api-Key` أو من `MANUS_API_KEY`.

</div>

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/api/health` | حالة الخادم |
| GET | `/api/config` | إعدادات الخادم (بدون كشف المفتاح) |
| GET | `/api/profiles` | ملفات الوكلاء واللغات المتاحة |
| POST | `/api/tasks` | إنشاء مهمة |
| GET | `/api/tasks` | قائمة المهام (`limit`, `cursor`, `order`, `scope`) |
| GET/PATCH/DELETE | `/api/tasks/{id}` | تفاصيل / تعديل / حذف |
| GET/POST | `/api/tasks/{id}/messages` | قراءة المحادثة / إرسال رسالة |
| POST | `/api/tasks/{id}/stop` | إيقاف المهمة |
| POST | `/api/tasks/{id}/confirm` | تأكيد إجراء معلّق |
| POST | `/api/files` | رفع ملف (`multipart/form-data`, الحقل `file`) |
| GET | `/api/files/{id}` | بيانات ملف |
| GET | `/api/usage`, `/api/credits` | سجل الاستخدام / الرصيد |
| GET | `/api/agents`, `/api/projects`, `/api/skills`, `/api/connectors`, `/api/browser-clients` | موارد الحساب |
| POST | `/api/projects` | إنشاء مشروع |
| GET/POST/DELETE | `/api/webhooks`, `/api/webhooks/{id}` | إدارة الـ Webhooks |
| POST | `/api/webhook` | مستقبل أحداث Manus (عام) |
| GET/DELETE | `/api/webhook/events` | الأحداث المخزّنة / تفريغها |

<div dir="rtl">

أمثلة كاملة (curl + JavaScript): [`docs/API.md`](docs/API.md).

</div>

---

## 6) التطوير والاختبار

```bash
make test         # go test ./... -race -cover
make agent-test   # 19 اختبارًا لمنطق الوكيل (Node)
make sync-assets  # نسخ واجهة الوكيل إلى أصول الأندرويد
make vet          # go vet
make fmt          # gofmt
make icons        # regenerate icons
make build        # bin/manusweb
```

<div dir="rtl">

الاختبارات تُشغّل خادم Manus وهميًا داخل العملية (httptest) وتتحقق من كل نقطة نهاية: بناء حمولة v2،
الترويسات، معالجة الأخطاء (401/400)، رفع الملفات، أحداث الـ Webhook، وخدمة ملفات الواجهة.

</div>

---

## License

<div dir="rtl">

الكود هنا برخصة MIT. مكتبة `manus-ai-go` مضَمَّنة كما هي برخصتها MIT (انظر `third_party/manus-ai-go/LICENSE`)،
وهي مشروع مجتمعي مستقل وغير مرتبط رسميًا بـ Manus.

</div>
