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

### النماذج المسجّلة + خاصية اختيار النماذج (مُختبَرة ✅)

| الموديل | الاختصار | الحالة |
| --- | --- | --- |
| `dahl-global/MiniMaxAI/MiniMax-M2.7` | `minimax` | الافتراضي |
| `dahl-global/zai-org/GLM-5.3-Flash` | `glm` | بديل #1 |
| `cleanapis-com/deepseek-v4-flash-0731` | `ds` | بديل #2 |
| `dahl-global/deepseek-ai/DeepSeek-V4-Flash-0731` | `ds-dahl` | إضافي |

**اختيار النموذج:**

```bash
bash deploy/wormagent/choose-model.sh          # قائمة مرقّمة تفاعلية (اكتب الرقم)
bash deploy/wormagent/choose-model.sh glm      # بالاختصار
bash deploy/wormagent/choose-model.sh 2        # بالرقم
bash deploy/wormagent/choose-model.sh --list   # عرض القائمة فقط
```

أو يدويًا: `node wormagent.mjs models set <alias|provider/model>` ثم إعادة تشغيل البوابة.
ولإدارة الاختصارات والبدائل: `models aliases add <alias> <model>` و`models fallbacks add|list|clear`.

> **ضبط السياق:** المزوّد المخصّص يُسجَّل افتراضيًا بـ `contextWindow: 16000` ويُنتج تحذير
> `low context window`. في هذا الإعداد رُفعت إلى `contextWindow: 64000` و`maxTokens: 8192`
> لكل الموديلات (عدّلها من `models.providers.<provider>.models` إن أردت).

### إضافة كل موديلات مزوّد بأمر واحد (مُختبَر ✅)

```bash
# عرض فقط (بدون أي تعديل)
bash deploy/wormagent/add-provider-models.sh https://codecraftapi.com/v1 "cc_xxx" codecraftapi --dry-run

# إضافة كل الموديلات الناقصة + إعادة تشغيل البوابة
bash deploy/wormagent/add-provider-models.sh https://codecraftapi.com/v1 "cc_xxx" codecraftapi
```

* يجلب `/models` من المزوّد، **يستبعد** المسجّل مسبقًا، يسجّل المزوّد، ثم يكتب كل الموديلات في
  `models.providers.<id>.models` (مع `--context N` لتغيير نافذة السياق)، ويُعيد الموديل الافتراضي كما كان.
* مثال حقيقي: أضاف **33 موديلًا** من `codecraftapi.com` (claude-opus-5 · gpt-5.6-luna · grok-4.6 ·
  gemini-3.7-flash · kimi-k3 · qwen3.8-max · seed-2.1-pro · muse-spark-1.1 …).

**سجل المزوّدين الحالي:**

| المزوّد | عدد الموديلات | حالة المفتاح |
| --- | --- | --- |
| `dahl-global` | 3 | ✅ يعمل (MiniMax-M2.7 · GLM-5.3-Flash · DeepSeek-V4-Flash) |
| `codecraftapi` | 33 | ⚠️ يرجّع `402 insufficient_quota` — يحتاج تفعيل الدفع من لوحة المزوّد |
| `cleanapis-com` | 1 | ⚠️ يرجّع `402 insufficient_quota` |

> الموديلات المسجّلة بمفاتيح بلا رصيد تبقى في الإعداد وتعمل **فورًا** بمجرد تفعيل الرصيد — بدون أي تعديل.
> والافتراضي حالياً `DeepSeek-V4-Flash` من `dahl-global` (شغّال) مع بدائل `GLM-5.3-Flash` ثم `MiniMax-M2.7`.

### ⚠️ لماذا كانت القائمة تعرض موديلًا واحدًا فقط؟ (سبب مهم)

فيه فرق جوهري بين مستويين:

| المستوى | المفتاح | الوظيفة |
| --- | --- | --- |
| موديلات المزوّد | `models.providers.<id>.models` | ما يستطيع النظام تشغيله |
| **الكتالوج المسموح** | `agents.defaults.models` | **ما يظهر للمستخدم** في `/models` وفي منتقي النماذج |

أمر `/models` يعرض **الكتالوج المسموح فقط**، لذلك بعد إضافة 33 موديلًا لم تظهر إلا في `models.providers`
ولم يظهر منها إلا واحد. الحل:

```bash
bash deploy/wormagent/sync-catalog.sh              # مزامنة كل المزوّدين (مُختبَر: يضيف الناقص فقط)
bash deploy/wormagent/sync-catalog.sh codecraftapi # مزوّد واحد
```

> سكربت `add-provider-models.sh` ينادي هذه المزامنة تلقائيًا بعد كل إضافة.
> ولمشاهدة كل شيء بدون إضافته للكتالوج: `/models <provider> all`.

### النماذج المطلوبة (مسجّلة مع اختصارات) ✅

| الموديل | الاختصار | الأمر من تلجرام |
| --- | --- | --- |
| `codecraftapi/claude-opus-5.5` | `opus55` | `/opus55` |
| `codecraftapi/claude-opus-5` | `opus5` | `/opus5` |
| `codecraftapi/kimi-k2.6` | `kimi` | `/kimi` |
| `codecraftapi/deepseek-v4-pro-0813` | `dspro` | `/dspro` |
| `codecraftapi/gpt-5.6-luna` | `luna` | `/luna` |
| `codecraftapi/claude-mythos-preview` | `mythos` | `/mythos` |

### خطأ `402 Billing verification failed` — لا يُعالَج تلقائيًا

* وثائق WormAgent تعتبر أخطاء الرصيد («insufficient credits» / «credit balance too low») **قابلة للتحويل**
  لموديل بديل، لكن الرسالة `Billing verification failed` (402) **لا تُصنَّف كذلك** في هذه النسخة
  (ظهر `failoverReason: null` في السجل) → تظهر للمستخدم كنص خطأ بدل التحويل للبديل.
* **لا يوجد حل من جهتنا:** يجب تفعيل الدفع/الرصيد في لوحة المزوّد (`codecraftapi.com` / `cleanapis.com`).
  وبمجرد التفعيل تعمل كل الموديلات المسجّلة فورًا بدون أي تعديل.
* للتجربة الآن: الافتراضي `DeepSeek-V4-Flash` + البدائل `GLM` ثم `MiniMax` (كلها من `dahl-global` وتعمل ✅).

### إرسال الصور وملفات PDF من الوكيل (مُختبَر ✅)

الوكيل يقدر يبعت ملفات بطريقتين:

1. **سطر `MEDIA:<path>` في ردّه** (الطريقة الرسمية): سطر مستقل، مثال:
   ```text
   تم إنشاء التقرير.
   MEDIA:/home/node/.wormagent/workspace/out/report.pdf
   ```
2. **أداة `message` أو الـ CLI عبر `exec`** (التي استخدمها الوكيل فعليًا):
   ```bash
   node ~/wormagent/wormagent.mjs message send --channel telegram --target <chatId> \
     --message "تقرير" --media /absolute/path/inside/workspace/out/report.pdf
   ```

**قيد إلزامي:** الملف لازم يكون **داخل مجلد العمل** — أي مسار في `/tmp` يُرفض:
`LocalMediaAccessError: Local media path is not under an allowed directory`.

**إنشاء PDF عربي (RTL):** أداة جاهزة في [`tools/make_pdf.py`](tools/make_pdf.py) + خط `Noto Naskh Arabic`:

```bash
pip install --trusted-host pypi.org --trusted-host files.pythonhosted.org \
  fpdf2 uharfbuzz arabic-reshaper python-bidi pillow
mkdir -p ~/.wormagent/fonts && curl -sSL -o ~/.wormagent/fonts/NotoNaskhArabic.ttf \
  'https://github.com/google/fonts/raw/main/ofl/notonaskharabic/NotoNaskhArabic%5Bwght%5D.ttf'

printf '%s\n' '# تقرير' '' '- نقطة أولى' '- نقطة ثانية' > out/report.txt
python3 tools/make_pdf.py out/report.pdf out/report.txt --title 'تقرير مختصر'
```

**إعدادات مرتبطة (مضبوطة):** `tools.allow: ["message"]` (أداة الإرسال) ·
`agents.defaults.pdfModel` (لقراءة PDF الوارد عبر أداة `pdf`).

**اختبار حقيقي:** الوكيل أنشأ `pyramids.pdf` ثم أرسله بنفسه → `✅ Sent via Telegram. Message ID: 81` ✅
(وصورة تجريبية: Message ID 73 ✅).

### لماذا توقّف الوكيل عن الرد؟ (`No reply from agent`)

* الموديل **MiniMax-M2.7** كان يُخرج `thinking` و`NO_REPLY` داخل النص، و`NO_REPLY` توكن «رد صامت»
  في WormAgent → الرسالة لا تُرسل أبدًا.
* الحل المطبّق: الافتراضي بقى **DeepSeek-V4-Flash** (ردود نظيفة)، والبدائل **GLM-5.3-Flash** ثم MiniMax.
* إضافة: تعليمات الوكيل في `AGENTS.md`/`TOOLS.md` (مجلد العمل) تشرح إنشاء PDF/الصور وطريقة الإرسال
  وقاعدة «المسارات داخل مجلد العمل فقط».

### 🎛️ لوحة التحكم `/panel` (مُختبَرة ✅)

زر في قائمة البوت (custom command) يعرض لك حالة كل شيء في رسالة واحدة:

```bash
bash deploy/wormagent/model-panel.sh            # فحص المفاتيح مباشرة (فحص حقيقي)
bash deploy/wormagent/model-panel.sh --no-probe # بدون فحص (أسرع)
```

تعرض اللوحة:

* 🧠 **الموديل الحالي** (الافتراضي) + كل البدائل.
* 🤖 **الوكلاء** وموديل كل وكيل ومساحة عمله.
* 🔌 **المزوّدون** وحالة كل مفتاح (فحص حقيقي: ✅ يعمل / ⚠️ بلا رصيد / ⚠️ مفتاح مرفوض…).
* 📦 **كل الموديلات** مع اختصاراتها، ومعلَّمة بـ `⭐ الافتراضي` و`🔁 بديل#n` و`🤖 اسم الوكيل`.

**كيف تعمل:** `channels.telegram.customCommands` تضيف الزر في القائمة، وتعليمات الوكيل في `AGENTS.md`
تجعل الوكيل ينفّذ السكربت ويرد بمخرجه كما هو.

```bash
node wormagent.mjs config set channels.telegram.customCommands \
  '[{"command":"panel","description":"لوحة التحكم: الموديل الحالي والوكلاء وحالة المفاتيح"}]'
cp deploy/wormagent/model-panel.sh ~/.wormagent/tools/ && chmod +x ~/.wormagent/tools/model-panel.sh
```

### ⚠️ خطأ خطير يجب تجنّبه: `tools.allow` يلغي كل الأدوات الأخرى

`tools.allow` ليس «إضافة» — بل **allow-only** (قائمة سماح حصرية). فإذا كتبت `tools.allow: ["message"]`
تُقفل **كل** الأدوات الأخرى (بما فيها `exec`) ويظهر للوكيل: `Tool exec not found` مع `capabilities=none`
فيصبح الوكيل عاجزًا عن أي تنفيذ. الصحيح هو سرد المجموعات المطلوبة كاملة:

```bash
node wormagent.mjs config set tools.allow \
  '["group:fs","group:runtime","group:sessions","group:memory","group:messaging"]'
```

> ملاحظة: `group:runtime` = `exec`+`process` · `group:fs` = `read/write/edit/apply_patch` ·
> `group:sessions` = `sessions_*` · `group:messaging` = `message` (لإرسال رسائل/ملفات).
> بعد أي تعديل على `tools.*` أو الموديلات: **أعد تشغيل البوابة**.

### أزرار تبديل النماذج داخل البوت (تلجرام) — مدمجة أصلًا ✅

WormAgent فيه لوحة أزرار Inline جاهزة، وما تحتاج أي كود إضافي — فقط تفعيل الإعدادات:

```bash
node wormagent.mjs config set channels.telegram.capabilities.inlineButtons all
node wormagent.mjs config set commands.native auto
node wormagent.mjs config set commands.text true
# ثم أعد تشغيل البوابة
bash deploy/wormagent/setup.sh --stop && bash deploy/wormagent/setup.sh --skip-install
```

**كيف تستخدمها من تلجرام:**

| ما ترسله | النتيجة |
| --- | --- |
| اضغط زر القائمة (☰ أو `/`) | تظهر 46 أمرًا منها **`/model`** و**`/models`** |
| `/models` | لوحة أزرار: **Select a provider** → اضغط المزوّد → أزرار الموديلات (مع ترقيم الصفحات) |
| `/model` | منتقي مرقّم + الموديل الحالي للوكيل |
| `/model status` | تفاصيل: الموديل + حالة المفتاح + `baseUrl` و`api` |
| `/model glm` | تبديل مباشر (بالاختصار أو `provider/model`) |
| `/minimax` · `/glm` · `/ds` · `/ds-dahl` | اختصارات مباشرة (Aliases مسجّلة كأوامر) |

> `/model ...` من تلجرام يبدّل **موديل الجلسة** (هذه المحادثة)، أما
> `deploy/wormagent/choose-model.sh` فيبدّل **الموديل الافتراضي العام** لكل المحادثات.

### الوكلاء (Agents): فرعيون + متعددون — مفعّلون ومُختبَرون ✅

**أ) الوكلاء الفرعيون (Sub-agents)** — مهام في الخلفية/بالتوازي:

| الإعداد | القيمة |
| --- | --- |
| `agents.defaults.subagents.maxSpawnDepth` | `2` (نمط المنسّق: رئيسي → منسّق → عمّال) |
| `maxChildrenPerAgent` / `maxConcurrent` | `5` / `8` |
| `runTimeoutSeconds` / `archiveAfterMinutes` | `900` / `60` |
| `subagents.model` | `dahl-global/zai-org/GLM-5.3-Flash` (أرخص/أسرع للوكلاء الفرعيين) |

من تلجرام: `/subagents list` · `/subagents spawn <agentId> <task>` · `/subagents info|log|steer|kill`

```text
/subagents spawn researcher اكتب تقريرًا قصيرًا عن الطاقة الشمسية
```

> اختبار حقيقي: الوكيل شغّل وكيلًا فرعيًا (`agent:main:subagent:<uuid>`) وأعلن نتيجته في المحادثة ✅

**ب) تعدد الوكلاء (Multi-agent)** — عقول منفصلة (مساحة عمل + جلسات + شخصية + موديل):

| الوكيل | الموديل | مساحة العمل |
| --- | --- | --- |
| `main` (الافتراضي) | `MiniMaxAI/MiniMax-M2.7` | `~/.wormagent/workspace` |
| `researcher` | `zai-org/GLM-5.3-Flash` | `~/.wormagent/workspace-researcher` |

```bash
node wormagent.mjs agents list                      # عرض الوكلاء
node wormagent.mjs agents add <name> --workspace <dir> --model <id> --non-interactive
node wormagent.mjs agents bind <name> --bind telegram:<accountId>   # توجيه قناة/حساب لوكيل
node wormagent.mjs agents delete <name>             # حذف وكيل
```

* `/agents` من تلجرام = الوكلاء المرتبطون بالـ thread الحالي.
* السماح للوكيل الرئيسي بالتفويض لأي وكيل: `agents.list[].subagents.allowAgents: ["*"]` (مضبوط).
* **تنبيه:** ملف الاعتماد `auth-profiles.json` **لكل وكيل على حدة**؛ ومزوّدونا المخصّصون مفاتيحهم
  في الإعداد العام (`models.providers`) فتصل لكل الوكلاء تلقائيًا.

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
| `402 insufficient_quota` / `Billing verification failed` | المفتاح صالح لكن الحساب بلا رصيد أو طريقة الدفع غير مُتحقَّقة — راجع لوحة المزوّد. الموديلات تبقى مسجّلة وتعمل فور تفعيل الرصيد. |
| `LocalMediaAccessError` عند إرسال ملف | الملف خارج مجلد العمل — انقله إلى `~/.wormagent/workspace/...` ثم أرسله. |
| `No reply from agent` (البوت صامت) | الموديل يُخرج `NO_REPLY` أو `thinking` داخل النص — بدّل لموديل أنظف: `node wormagent.mjs models set ds-dahl`. |
| `Not enough horizontal space` في fpdf2 | استخدم عرض صفحة صريح (`pdf.epw`) و`new_x/new_y` — الأداة `tools/make_pdf.py` تفعل ذلك. |
| `Unknown model` بعد إضافة مزوّد جديد | تأكد أن الافتراضي لم يتغيّر لأول موديل جديد: `node wormagent.mjs models set minimax` (سكربت الإضافة يُعيده تلقائيًا). |
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

