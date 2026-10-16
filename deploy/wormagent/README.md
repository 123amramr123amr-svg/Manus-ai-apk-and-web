# WormAgent — تشغيل البوابة والواجهة، وحقيقة «الـ APK»

<div dir="rtl">

## 1) ما هو ريبو `BlackXV2vip/wormagent`؟

| العنصر | الحالة |
| --- | --- |
| النوع | نسخة **مبنية مسبقًا** من WormAgent `2026.3.19` (بوابة وكيل ذكاء اصطناعي متعددة القنوات، Node/TS، رخصة MIT، تفريع من OpenClaw) |
| الموجود فعلًا | `wormagent.mjs` (المشغّل) + `dist/` (كود JS مبني، 33MB) + `extensions/` (75 إضافة) + `skills/` + `docs/` + `package.json` |
| **غير الموجود** | لا مصدر TypeScript (`src/`)، **ولا مصدر أندرويد (`apps/android`)، ولا أي ملف APK** |

⚠️ **تنبيه قبل الاستخدام:** هذا **نشر غير رسمي** لحزمة مبنية مسبقًا من حساب فردي (بدون مصدر).
شغّله على جهاز/VM مستقل ولا تضع فيه مفاتيح لا تتحمّل تسريبها، لأن الوكيل نفسه يستطيع تنفيذ أوامر
على الجهاز وفتح متصفح وقراءة ملفات (هذه وظيفته لا خلل فيه — لكن يعني: لا تشغّله على جهازك الأساسي).

---

## 2) التشغيل بأمر واحد (مُختبَر فعليًا)

```bash
# من داخل هذا المستودع
bash deploy/wormagent/setup.sh

# أو مع قناة تلجرام + فتح الوصول للشبكة المحلية
bash deploy/wormagent/setup.sh --telegram "123456:AA..." --bind lan
```

ماذا يفعل السكربت؟

1. يتحقق من Node.js (يحتاج **22.16+**، والموصى به 24).
2. يستنسخ `https://github.com/BlackXV2vip/wormagent.git` إلى `~/wormagent` (أو يحدّثه إن وُجد).
3. يثبّت الاعتماديات: `npm install --omit=dev --legacy-peer-deps` (≈485 حزمة، ≈560MB).
4. يكتب الإعدادات في `~/.wormagent/wormagent.json` (وضع `local`، منفذ `18789`، مصادقة بتوكن عشوائي).
5. يشغّل البوابة في الخلفية ويطبع رابط الواجهة + مفتاح الدخول.

| الخيار | الوظيفة |
| --- | --- |
| `--dir <path>` | مجلد الكود (افتراضي `~/wormagent`) |
| `--port <n>` | منفذ البوابة (افتراضي `18789`) |
| `--bind loopback\|lan\|tailnet` | نطاق الاستماع (افتراضي `loopback`) |
| `--telegram "<TOKEN>"` | تفعيل قناة تلجرام بمفتاح BotFather |
| `--no-start` | إعداد بدون تشغيل |
| `--skip-install` | تخطّي `npm install` |
| `--stop` | إيقاف البوابة التي شغّلها السكربت |

---

## 3) الواجهة — نفس واجهة المشروع

* **Control UI** على: `http://127.0.0.1:18789/` (نفس واجهة WormAgent: Chat، Channels، Sessions، Cron، Skills، Nodes، Config).
* المصادقة: توكن البوابة (يطبعه السكربت) — الصقه في شاشة الإعدادات عند أول اتصال.
* من الجوال: افتح نفس الرابط عبر الشبكة المحلية/التابلت، ثم **«إضافة إلى الشاشة الرئيسية»** فتعمل كتطبيق (PWA).
* الواجهة مترجمة إلى: `en, zh-CN, zh-TW, pt-BR, de, es` — **لا يوجد عربي** حتى الآن.
* المنافذ المرافقة: `18791` تحكّم المتصفح، و`/__wormagent__/canvas/` مساحة Canvas.

> **الوصول من الجوال:** الأفضل هو **Tailscale** (يعطي HTTPS وشبكة خاصة) ثم افتح
> `https://<host>.<tailnet>.ts.net:18789` — لأن الواجهة تشترط سياقًا آمنًا واعتمادًا للأجهزة الجديدة.
> أما عبر HTTP على الشبكة المحلية فتحتاج `gateway.controlUi.allowedOrigins` (يفعله السكربت مع `--bind lan`)
> وقد يُطلب اعتماد الجهاز: `node wormagent.mjs devices list` ثم `devices approve <requestId>`
> (`allowInsecureAuth` يعمل مع localhost فقط ولا يُرخي شروط الأجهزة البعيدة).

</div>

<div dir="rtl">

## 4) تلجرام (من @BotFather)

```bash
bash deploy/wormagent/setup.sh --telegram "123456:AAxxxxxxxxxxxxxxxxxxxx"
```

1. افتح تلجرام → **@BotFather** → `/newbot` → انسخ التوكن.
2. شغّل الأمر أعلاه (يكتب التوكن في `channels.telegram.botToken` ويفعّل القناة).
3. أرسل رسالة للبوت من حسابك، ثم اعتمد الاقتران:

```bash
cd ~/wormagent
node wormagent.mjs pairing list telegram
node wormagent.mjs pairing approve telegram <CODE>
```

**ما تم التحقق منه فعليًا:** القناة تُقلع وتتصل بـ `api.telegram.org`؛ وبمفتاح تجريبي غير صحيح ظهر
`401: Unauthorized` — أي أن كل شيء سليم ولا ينقص إلا مفتاح BotFather حقيقي.

> للمجموعات: الإعداد الافتراضي `groupPolicy: allowlist` يمنع رسائل المجموعات حتى تضيف معرّفات
> المرسلين في `channels.telegram.groupAllowFrom` (أو `allowFrom`)، أو تجعل السياسة `open`.

## 5) موديل المحادثة (LLM) — إلزامي

البوابة تعمل وتُظهر الواجهة بدون مفتاح موديل، لكن **لن تجيب على أي رسالة** حتى تضيف مزوّدًا:

```bash
cd ~/wormagent
node wormagent.mjs onboard          # معالج تفاعلي: اختر المزوّد والصق المفتاح
# أو غير الافتراضي (anthropic/claude-opus-4-6) لأي مزوّد مدعوم:
node wormagent.mjs config set agent.model "openai/gpt-5.2"
```

إضافة مفتاح بدون تفاعل (مثال Anthropic):

```bash
node wormagent.mjs onboard --non-interactive --mode local \
  --auth-choice apiKey --anthropic-api-key "sk-ant-..." \
  --secret-input-mode plaintext --gateway-port 18789 --gateway-bind loopback
```

بدون مفتاح يظهر هذا الخطأ عند أول رسالة (مُتحقَّق منه):

```text
FailoverError: No API key found for provider "anthropic".
Auth store: ~/.wormagent/agents/main/agent/auth-profiles.json
```

### مزوّد مخصّص متوافق مع OpenAI (مُختبَر ✅)

```bash
bash deploy/wormagent/add-model-key.sh custom "cc-xxxxxxxx" "https://cleanapis.com/v1" "deepseek-v4-flash-0731"
```

* السكربت يستنتج معرّف المزوّد من الرابط، وWormAgent **يحوّل النقاط والرموز إلى شرطات**:
  `cleanapis.com` ← المعرّف `cleanapis-com` ← الموديل `cleanapis-com/deepseek-v4-flash-0731`.
* **تنبيهان مهمّان (بالتجربة):**
  1. لا تضبط الموديل بـ `config set agent.model` — هذا المفتاح انتقل إلى `agents.defaults.model.primary`
     ويعطي `Config validation failed`. استخدم: `node wormagent.mjs models set <provider>/<model>`.
  2. لو ظهر `Unknown model: ...` فراجع `node wormagent.mjs models list` وخُذ المعرّف الظاهر فيه بالضبط.

المزوّدات المدعومة تشمل: Anthropic، OpenAI، Google، Bedrock، OpenRouter، Mistral، Moonshot،
Z.AI، xAI، Ollama (محلي بلا مفتاح) وغيرها عبر الإضافات.

## 6) موضوع الـ APK — الحقيقة والخيارات

**لا يمكن بناء APK من ريبو `BlackXV2vip/wormagent`:** لا يحتوي أي مصدر أندرويد (لا `apps/android`
ولا Gradle)، ولا يوجد فيه ملف APK جاهز. (كما أن بيئة التنفيذ هنا بلا JDK/Android SDK.)

الخيارات العملية:

| الخيار | التفاصيل |
| --- | --- |
| **PWA (الأسهل الآن)** | افتح رابط الواجهة من متصفح الجوال → «إضافة إلى الشاشة الرئيسية» → أيقونة وشاشة كاملة. |
| **APK غلاف WebView** | مشروع `android/` في هذا المستودع يفتح أي رابط خادم داخل WebView. ادمج رابط البوابة (`-PmanusServerUrl=...`) وابنِه عبر `ci/github-workflows/build-apk.yml` أو `android/build-manual.sh`. **يوجد APK مبني مسبقًا في مستودعك:** `manus-ai-1.1.0-debug.apk` (صفحة Releases) — ثبّته ثم اكتب عنوان البوابة في شاشة «عنوان الخادم». |
| **تطبيق أندرويد الرسمي** | تطبيق «OpenClaw Android» (عقدة مرافقة للبوابة: محادثة/صوت/موافقات/شاشة) مصدره `openclaw/openclaw` تحت `apps/android`، ويُوزَّع على **Google Play**. بناؤه من المصدر يحتاج Android Studio/SDK أو GitHub Actions. |

> الخلاصة: نفس الواجهة تعمل اليوم على الجوال عبر المتصفح/WebView بلا أي APK خاص، والـ APK الرسمي
> يحتاج مصدر OpenClaw لا هذا الريبو.

## 7) الأمان (اقرأ هذا)

* لا تفتح البوابة على الإنترنت العام مباشرة. استخدم **Tailscale** (الطريقة الموصى بها في الوثائق) أو توكن قوي + `gateway.controlUi.allowedOrigins`.
* الاقتران (pairing) مفروض افتراضيًا: أي جهاز/مرسل جديد يحتاج موافقة (`wormagent devices list` / `pairing approve`).
* الوكيل ينفّذ أدوات (shell/ملفات/متصفح) على الجهاز → شغّله في حاوية أو VM، مع مستخدم غير جذري.
* المفاتيح في `~/.wormagent/wormagent.json` → اجعل صلاحياته `600`.
* لتشغيله كخدمة دائمة: انسخ `deploy/wormagent/wormagent.service` إلى `/etc/systemd/system/`.

## 8) استكشاف الأخطاء

| المشكلة | الحل |
| --- | --- |
| `npm error ERESOLVE` | استخدم `npm install --omit=dev --legacy-peer-deps` (السكربت يفعله تلقائيًا). |
| البوابة لا تقلع | يجب `gateway.mode: "local"` في الإعدادات (أو `wormagent gateway --allow-unconfigured`). |
| `401 Unauthorized` في تلجرام | توكن BotFather غير صحيح/منسوخ ناقصًا. |
| الواجهة لا تجيب، أو `pairing required` | اعتمد الجهاز: `node wormagent.mjs devices list` ثم `devices approve <requestId>` (اتصالات 127.0.0.1 تُعتمد تلقائيًا). |
| لا رد على الرسائل | لا يوجد مزوّد موديل/مفتاح — نفّذ `node wormagent.mjs onboard` أو `bash deploy/wormagent/add-model-key.sh ...`. |
| `Unknown model: ...` | معرّف المزوّد مختلف عمّا توقّعت (النقاط تصبح شرطات) — راجع `node wormagent.mjs models list` وخُذ المعرّف الظاهر. |
| `Config validation failed: agent.* was moved` | استخدم `node wormagent.mjs models set` بدل `config set agent.model`. |
| `extension entry escapes package directory` | تحذير غير ضار في هذه الحزمة المبنية مسبقًا (17 إضافة) ولا يمنع التشغيل. |
| الوصول من جهاز آخر مرفوض | أضف أصل المتصفح في `gateway.controlUi.allowedOrigins` (السكربت يفعله مع `--bind lan`). |
| تريد معرفة الحالة | `node wormagent.mjs doctor` و`node wormagent.mjs gateway health`. |

---

## English quick summary

* `BlackXV2vip/wormagent` is a **prebuilt-only** republish of WormAgent 2026.3.19: launcher + `dist/`
  bundles + 75 extensions. **No TypeScript sources, no Android sources, no APK.**
* `bash deploy/wormagent/setup.sh` clones it, installs deps, writes `~/.wormagent/wormagent.json`
  (local mode, port 18789, token auth) and starts the Gateway — verified in this repo.
* Control UI (same interface as the project) is served at `http://127.0.0.1:18789/`.
* Telegram works with a BotFather token (`--telegram "123:ABC"`), verified against `api.telegram.org`.
* An LLM provider key is required for chat replies (`node wormagent.mjs onboard`).
* For mobile: use the Control UI as a PWA, or wrap it in this repo's WebView APK. The official
  Android companion app lives in `openclaw/openclaw/apps/android` (distributed via Google Play).

</div>

